// Package nativesrv implements TCP server that uses RCS Native Protocol (RCSP)
// as its application layer protocol.
//
// See RCSP specification at https://github.com/nmezhenskyi/rcs/blob/main/api/native/rcs.md.
package nativesrv

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nmezhenskyi/rcs/internal/cache"
	"github.com/rs/zerolog"
)

const (
	MaxMessageSize     = 1048576 // 1 MB
	DefaultMessageSize = MaxMessageSize

	shutdownPollIntervalMax = 500000000 // 500ms
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
	if s.inShutdown.isSet() {
		s.Logger.Info().Msg("ListenAndServeTLS aborted: Server is in shutdown mode")
		return nil
	}
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

// ListenAndServeTLS listens on the given TCP network address addr and
// handles requests on incoming TLS connections according to RCSP.
//
// Requires valid certiticate and key files containing PEM encoded data.
func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	if s.inShutdown.isSet() {
		s.Logger.Info().Msg("ListenAndServeTLS aborted: Server is in shutdown mode")
		return nil
	}
	s.Logger.Info().Msg("Starting tls native server on " + addr)
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		s.Logger.Error().Err(err).Msg("failed to load tls certificate")
		return err
	}
	tlsConfig := tls.Config{
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
		Certificates: []tls.Certificate{cert},
	}
	listener, err := tls.Listen("tcp", addr, &tlsConfig)
	if err != nil {
		s.Logger.Error().Err(err).Msg("failed to start tls listener")
		return err
	}
	err = s.serve(listener)
	if err != nil {
		s.Logger.Error().Err(err).Msg("failed while serving")
	}
	return err
}

// Shutdown gracefully shuts down the server without interrupting any
// active connections. Waits until all connections are closed or until context
// timeout runs out.
//
// Polling strategy taken from http.Server.Shutdown():
// https://pkg.go.dev/net/http#Server.Shutdown.
func (s *Server) Shutdown(ctx context.Context) error {
	s.inShutdown.setTrue()
	err := s.listener.Close()
	if err != nil {
		s.Logger.Error().Err(err).Msg("underlying tcp listener errored while closing")
	}

	pollIntervalBase := time.Millisecond
	nextPollInterval := func() time.Duration {
		// Add 10% jitter.
		interval := pollIntervalBase + time.Duration(rand.Intn(int(pollIntervalBase/10)))
		// Double and clamp for next time.
		pollIntervalBase *= 2
		if pollIntervalBase > shutdownPollIntervalMax {
			pollIntervalBase = shutdownPollIntervalMax
		}
		return interval
	}

	timer := time.NewTimer(nextPollInterval())
	defer func() {
		timer.Stop()
		s.Logger.Info().Msg("native server has been shutdown")
	}()
	for {
		if s.numConns() == 0 {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			timer.Reset(nextPollInterval())
		}
	}
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

	s.Logger.Info().Msg("native server has been closed")
	return err
}

// serve accepts connections on the given listener and delegates them to
// handleConnection for processing.
func (s *Server) serve(lis net.Listener) error {
	lis = &srvListener{Listener: lis}
	s.mu.Lock()
	s.listener = lis.(*srvListener)
	s.mu.Unlock()
	defer s.listener.Close()
	for {
		if s.inShutdown.isSet() {
			return nil
		}
		conn, err := lis.Accept()
		if err != nil {
			if !s.inShutdown.isSet() {
				s.Logger.Error().Err(err).Msg("failed to accept connection")
			}
			if _, ok := err.(net.Error); ok {
				continue
			}
			return err
		}
		s.Logger.Debug().Msg("Received new connection (" + conn.RemoteAddr().String() + ")")
		s.mu.Lock()
		s.activeConns[conn] = struct{}{}
		s.mu.Unlock()
		go s.handleConnection(conn)
	}
}

// handleConnection exchanges messages with the given connection. It processes an
// incoming request and sends a response according to RCSP. It can handle many
// sequential requests on a single connection. It is encouraged to reuse the same
// connection for multiple requests.
func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		s.mu.Lock()
		delete(s.activeConns, conn)
		s.mu.Unlock()
		s.Logger.Debug().Msg("Closed connection (" + conn.RemoteAddr().String() + ")")
	}()

MsgLoop:
	for {
		buf := make([]byte, DefaultMessageSize)
		n, err := conn.Read(buf)
		if n == 0 || err != nil {
			s.Logger.Error().Err(err).Msg(fmt.Sprintf("error while reading from %s", conn.RemoteAddr()))
			return
		}

		req, err := parseRequest(buf[:n])
		if err != nil {
			s.handleParsingError(conn, err)
			continue MsgLoop
		}

		switch string(req.command) {
		case "SET":
			s.handleSet(conn, &req)
		case "GET":
			s.handleGet(conn, &req)
		case "DELETE":
			s.handleDelete(conn, &req)
		case "PURGE":
			s.handlePurge(conn, &req)
		case "LENGTH":
			s.handleLength(conn, &req)
		case "KEYS":
			s.handleKeys(conn, &req)
		case "PING":
			s.handlePing(conn, &req)
		case "CLOSE":
			s.handleCloseConn(conn, &req)
			break MsgLoop
		default:
			s.handleInvalidCommand(conn, &req)
		}
	}
}

func (s *Server) handleSet(conn net.Conn, req *request) {
	var resp = response{}

	if len(req.key) == 0 {
		resp.writeError(conn, []byte("SET"), []byte("Key is missing"))
		return
	}
	if len(req.value) == 0 {
		resp.writeErrorWithKey(conn, []byte("SET"), []byte("Value is missing"), req.key)
		return
	}

	s.cache.Set(string(req.key), req.value)
	resp.command = []byte("SET")
	resp.ok = true
	resp.key = req.key
	resp.write(conn)
}

func (s *Server) handleGet(conn net.Conn, req *request) {
	var resp = response{}

	if len(req.key) == 0 {
		resp.writeError(conn, []byte("GET"), []byte("Key is missing"))
		return
	}
	if len(req.value) != 0 {
		resp.writeErrorWithKey(conn, []byte("GET"), []byte("Received unexpected value"), req.key)
		return
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
}

func (s *Server) handleDelete(conn net.Conn, req *request) {
	var resp = response{}

	if len(req.key) == 0 {
		resp.writeError(conn, []byte("DELETE"), []byte("Key is missing"))
		return
	}
	if len(req.value) != 0 {
		resp.writeErrorWithKey(conn, []byte("DELETE"), []byte("Received unexpected value"), req.key)
		return
	}

	s.cache.Delete(string(req.key))
	resp.command = []byte("DELETE")
	resp.ok = true
	resp.key = req.key
	resp.write(conn)
}

func (s *Server) handlePurge(conn net.Conn, req *request) {
	var resp = response{}
	s.cache.Purge()
	resp.command = []byte("PURGE")
	resp.ok = true
	resp.write(conn)
}

func (s *Server) handleLength(conn net.Conn, req *request) {
	var resp = response{}
	length := s.cache.Length()
	resp.command = []byte("LENGTH")
	resp.ok = true
	resp.value = []byte(strconv.Itoa(length))
	resp.write(conn)
}

func (s *Server) handleKeys(conn net.Conn, req *request) {
	var resp = response{}
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
}

func (s *Server) handlePing(conn net.Conn, req *request) {
	var resp = response{}
	resp.command = []byte("PING")
	resp.ok = true
	resp.message = []byte("PONG")
	resp.write(conn)
}

func (s *Server) handleCloseConn(conn net.Conn, req *request) {
	var resp = response{}
	resp.command = []byte("CLOSE")
	resp.ok = true
	resp.write(conn)
}

func (s *Server) handleInvalidCommand(conn net.Conn, req *request) {
	var resp = response{}
	resp.ok = false
	resp.message = []byte("Received invalid command")
	resp.write(conn)
}

func (s *Server) handleParsingError(conn net.Conn, parsingErr error) {
	s.Logger.Error().Err(parsingErr).
		Msg(fmt.Sprintf("error while parsing request from %s", conn.RemoteAddr()))
	var resp = response{}
	switch parsingErr {
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
}

func (s *Server) numConns() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.activeConns)
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
