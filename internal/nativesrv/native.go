// Package nativesrv implements TCP server that uses RCS Native Protocol (RCSP)
// as its application layer protocol.
//
// See RCSP specification at https://github.com/nmezhenskyi/rcs/blob/main/api/native/rcs.md.
package nativesrv

import (
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

func (s *Server) ListenAndServe(addr string) error {
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

func (s *Server) Shutdown() {
	// TODO: wait for all active conns to close and then gracefully shutdown the server
}

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
			s.Logger.Error().Err(err).Msg("connection read error")
			return
		}

		req, err := parseRequest(buf[:n])
		if err != nil {
			switch err {
			// TODO: handle errors
			}
		}

		switch string(req.command) {
		case "SET":
			resp.command = []byte("SET")
			if len(req.key) == 0 {
				resp.ok = false
				resp.message = []byte("Key is missing")
				resp.write(conn)
				continue MsgLoop
			}
			if len(req.value) == 0 {
				resp.ok = false
				resp.message = []byte("Value is missing")
				resp.write(conn)
				continue MsgLoop
			}
			s.cache.Set(string(req.key), req.value)
			resp.ok = true
			resp.key = req.key
			resp.write(conn)
		case "GET":
			resp.command = []byte("GET")
			if len(req.key) == 0 {
				resp.ok = false
				resp.message = []byte("Key is missing")
				resp.write(conn)
				continue MsgLoop
			}
			val, ok := s.cache.Get(string(req.key))
			resp.ok = ok
			resp.key = req.key
			resp.value = val
			if !resp.ok {
				resp.message = []byte("Not found")
			}
			resp.write(conn)
		case "DELETE":
			resp.command = []byte("DELETE")
			if len(req.key) == 0 {
				resp.ok = false
				resp.message = []byte("Key is missing")
				resp.write(conn)
				continue MsgLoop
			}
			s.cache.Delete(string(req.key))
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
			resp.message = []byte("Malformed request, invalid command")
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
