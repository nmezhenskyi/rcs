// Package nativesrv implements TCP server that uses RCS Native Protocol (RCSP)
// as its application layer protocol.
//
// See RCSP specification at https://github.com/nmezhenskyi/rcs/blob/main/api/native/rcs.md.
package nativesrv

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/nmezhenskyi/rcs/internal/cache"
	"github.com/rs/zerolog"
)

const (
	MaxMessageSize     = 1048576 // 1 MB
	DefaultMessageSize = MaxMessageSize
)

// Server implements RCS Native TCP Protocol.
type Server struct {
	cache *cache.CacheMap

	inShutdown atomicBool

	mu          sync.Mutex
	listener    *srvListener
	activeConns map[net.Conn]struct{}

	Logger zerolog.Logger // By defaut Logger is disabled, but can be manually attached.
}

// NewServer initializes a new Server instance ready to be used and returns a pointer to it.
// You can also attach a Logger to returned Server by accessing public field Server.Logger.
func NewServer(c *cache.CacheMap) *Server {
	if c == nil {
		c = cache.NewCacheMap()
	}
	return &Server{
		cache:       c,
		activeConns: make(map[net.Conn]struct{}),
		Logger:      zerolog.New(os.Stderr).Level(zerolog.Disabled),
	}
}

// ListenAndServe listens on the given TCP network address addr and
// handles requests on incoming connections according to RCSP.
func (s *Server) ListenAndServe(addr string) error {
	s.Logger.Info().Msg("Starting native server on " + addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.Logger.Error().Err(err).Msg("failed to start listener")
		return err
	}
	err = s.serve(listener)
	if err != nil {
		s.Logger.Error().Err(err).Msg("failed while serving")
	}
	return err
}

// TODO:
func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	return nil
}

// Shutdown gracefully shuts down the server without interrupting any
// active connections.
func (s *Server) Shutdown(ctx context.Context) error {
	s.inShutdown.setTrue()

	// TODO: wait for all active conns to close and then gracefully shutdown the server

	s.Logger.Info().Msg("native server has been shutdown")

	return nil
}

// Close immediately closes all active connections and underlying listener.
// For a graceful shutdown, use Shutdown.
//
// Close returns any error returned from closing the Server's underlying listener.
func (s *Server) Close() error {
	s.inShutdown.setTrue()
	err := s.listener.Close()
	if err != nil {
		s.Logger.Error().Err(err).Msg("underlying tcp listener errored while closing")
	}

	s.mu.Lock()
	for c := range s.activeConns {
		c.Close()
		delete(s.activeConns, c)
	}
	s.mu.Unlock()

	return err
}

// serve accepts connections on the given listener and delegates them to
// handleConnection for processing.
func (s *Server) serve(lis net.Listener) error {
	lis = &srvListener{Listener: lis}
	defer lis.Close()
	s.mu.Lock()
	s.listener = lis.(*srvListener)
	s.mu.Unlock()
	for {
		conn, err := lis.Accept()
		if err != nil {
			s.Logger.Error().Err(err).Msg("failed to accept connection")
			continue
		}
		s.mu.Lock()
		s.activeConns[conn] = struct{}{}
		s.mu.Unlock()
		go s.handleConnection(conn)
	}
}

// handleConnection exchanges messages with the given connection. It processes an
// incoming request and sends a response according to RCSP. It can handle many
// requests on a single connection. It is encouraged to reuse the same connection for
// multiple requests.
func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		s.mu.Lock()
		delete(s.activeConns, conn)
		s.mu.Unlock()
	}()

MsgLoop:
	for {
		var resp = response{}

		buf := make([]byte, DefaultMessageSize)
		n, err := conn.Read(buf)
		if n == 0 || err != nil {
			s.Logger.Error().Err(err).Msg(
				fmt.Sprintf("error while reading from %s", conn.RemoteAddr()))
			return
		}

		req, err := parseRequest(buf[:n])
		if err != nil {
			s.Logger.Error().Err(err).Msg(
				fmt.Sprintf("error while parsing request from %s", conn.RemoteAddr()))
			switch err {
			case ErrMalformedRequest:
				resp.writeError(conn, nil, []byte("Malformed request"))
			case ErrUnknownProtocol:
				resp.writeError(conn, nil, []byte("Unknown protocol"))
			case ErrInvalidKey:
				resp.writeError(conn, nil, []byte("Received invalid key"))
			case ErrInvalidValue:
				resp.writeError(conn, nil, []byte("Received invalid value"))
			default:
				resp.writeError(conn, nil, []byte("Unexpected error while parsing request"))
			}
			continue MsgLoop
		}

		switch string(req.command) {
		case "SET":
			if len(req.key) == 0 {
				resp.writeError(conn, []byte("SET"), []byte("Key is missing"))
				continue MsgLoop
			}
			if len(req.value) == 0 {
				resp.writeErrorWithKey(conn, []byte("SET"), []byte("Value is missing"), req.key)
				continue MsgLoop
			}
			s.cache.Set(string(req.key), req.value)
			resp.command = []byte("SET")
			resp.ok = true
			resp.key = req.key
			resp.write(conn)
		case "GET":
			if len(req.key) == 0 {
				resp.writeError(conn, []byte("GET"), []byte("Key is missing"))
				continue MsgLoop
			}
			if len(req.value) != 0 {
				resp.writeErrorWithKey(conn, []byte("GET"), []byte("Received unexpected value"), req.key)
				continue MsgLoop
			}
			val, ok := s.cache.Get(string(req.key))
			resp.command = []byte("GET")
			resp.ok = ok
			resp.key = req.key
			resp.value = val
			if !resp.ok {
				resp.message = []byte("Not found")
			}
			resp.write(conn)
		case "DELETE":
			if len(req.key) == 0 {
				resp.writeError(conn, []byte("DELETE"), []byte("Key is missing"))
				continue MsgLoop
			}
			if len(req.value) != 0 {
				resp.writeErrorWithKey(conn, []byte("DELETE"), []byte("Received unexpected value"), req.key)
				continue MsgLoop
			}
			s.cache.Delete(string(req.key))
			resp.command = []byte("DELETE")
			resp.ok = true
			resp.key = req.key
			resp.write(conn)
		case "PURGE":
			s.cache.Purge()
			resp.command = []byte("PURGE")
			resp.ok = true
			resp.write(conn)
		case "LENGTH":
			length := s.cache.Length()
			resp.command = []byte("LENGTH")
			resp.ok = true
			resp.value = []byte(strconv.Itoa(length))
			resp.write(conn)
		case "KEYS":
			resp.command = []byte("KEYS")
			keys := s.cache.Keys()
			if len(keys) != 0 {
				resp.ok = true
				resp.value = []byte(strings.Join(keys, ","))
			} else {
				resp.ok = false
				resp.message = []byte("No keys")
			}
			resp.write(conn)
		case "PING":
			resp.command = []byte("PING")
			resp.ok = true
			resp.write(conn)
		case "CLOSE":
			resp.command = []byte("CLOSE")
			resp.ok = true
			resp.write(conn)
			break MsgLoop
		default:
			resp.ok = false
			resp.message = []byte("Received invalid command")
			resp.write(conn)
		}
	}
}

func (s *Server) shuttingDown() bool {
	return s.inShutdown.isSet()
}

// srvListener wraps a net.Listener to protect it from multiple Close() calls.
type srvListener struct {
	net.Listener
	once     sync.Once
	closeErr error
}

func (l *srvListener) Close() error {
	if l == nil {
		return nil
	}
	l.once.Do(l.close)
	return l.closeErr
}

func (l *srvListener) close() {
	l.closeErr = l.Listener.Close()
}

type atomicBool int32

func (b *atomicBool) isSet() bool {
	return atomic.LoadInt32((*int32)(b)) != 0
}

func (b *atomicBool) setTrue() {
	atomic.StoreInt32((*int32)(b), 1)
}

func (b *atomicBool) setFalse() {
	atomic.StoreInt32((*int32)(b), 0)
}
