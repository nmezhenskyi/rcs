package nativesrv

import (
	"bytes"
	"net"
	"os"
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

// --- Public API: --- //

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

}

func (s *Server) Close() {
	s.inShutdown.setTrue()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listener.Close()
	for c := range s.activeConns {
		c.Close()
		delete(s.activeConns, c)
	}
}

// --- Private: --- //

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
	defer conn.Close()

	for {
		buf := make([]byte, DefaultMessageSize)
		n, err := conn.Read(buf)
		if n == 0 || err != nil {
			s.Logger.Error().Err(err).Msg("connection read error")
			return
		}

		tokens := bytes.SplitN(buf[:n], []byte("\r\n"), 4)
		if len(tokens) == 0 {
			conn.Write([]byte("RCSP/1.0 NOT_OK\r\nMESSAGE: Bad request\r\n"))
			continue
		}
		headerTokens := bytes.Split(tokens[0], []byte(" "))
		if len(headerTokens) != 2 || bytes.Compare(headerTokens[0], []byte("RCSP/1.0")) != 0 {
			conn.Write([]byte("RCSP/1.0 NOT_OK\r\nMESSAGE: Unknown protocol, expected RCSP/1.0\r\n"))
			continue
		}

		command := strings.TrimSuffix(string(headerTokens[1]), "\r\n")
		switch command {
		case "GET":

		case "SET":

		case "DELETE":

		case "PURGE":

		case "LENGTH":

		case "KEYS":

		case "PING":
			conn.Write([]byte("RCSP/1.0 PING OK\r\n"))
		default:
			conn.Write([]byte("RCSP/1.0 NOT_OK\r\nMESSAGE: Bad request\r\n"))
		}
	}
}

func (s *Server) shuttingDown() bool {
	return s.inShutdown.isSet()
}

// --- Helpers: --- //

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
