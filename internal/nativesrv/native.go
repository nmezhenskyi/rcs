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
	defer func() {
		conn.Close()
		s.mu.Lock()
		delete(s.activeConns, conn)
		s.mu.Unlock()
	}()

MsgLoop:
	for {
		buf := make([]byte, DefaultMessageSize)
		n, err := conn.Read(buf)
		if n == 0 || err != nil {
			s.Logger.Error().Err(err).Msg("connection read error")
			return
		}

		tokens := bytes.SplitN(buf[:n], []byte("\r\n"), 3)
		if len(tokens) == 0 {
			conn.Write([]byte("RCSP/1.0 NOT_OK\r\nMESSAGE: Malformed request\r\n"))
			continue
		}
		headerTokens := bytes.Split(tokens[0], []byte(" "))
		if len(headerTokens) != 2 || bytes.Compare(headerTokens[0], []byte("RCSP/1.0")) != 0 {
			conn.Write([]byte("RCSP/1.0 NOT_OK\r\nMESSAGE: Unknown protocol, expected RCSP/1.0\r\n"))
			continue
		}

		command := strings.TrimSuffix(string(headerTokens[1]), "\r\n")
		switch command {
		case "SET":
			if len(tokens) != 3 {
				conn.Write([]byte("RCSP/1.0 SET NOT_OK\r\nMESSAGE: Key and/or value are missing\r\n"))
				continue MsgLoop
			}
			keyTokens := bytes.SplitN(tokens[1], []byte(": "), 2)
			valueTokens := bytes.SplitN(tokens[2], []byte(": "), 2)
			if len(keyTokens) != 2 || len(valueTokens) != 2 {
				conn.Write([]byte("RCSP/1.0 SET NOT_OK\r\nMESSAGE: Invalid key and/or value format\r\n"))
				continue MsgLoop
			}
			s.cache.Set(string(keyTokens[1]), valueTokens[1])
			response := append([]byte("RCSP/1.0 SET OK\r\nKEY: "), keyTokens[1]...)
			response = append(response, []byte("\r\n")...)
			conn.Write(response)
		case "GET":
			if len(tokens) != 2 {
				conn.Write([]byte("RCSP/1.0 SET NOT_OK\r\nMESSAGE: Key is missing\r\n"))
				continue MsgLoop
			}
			keyTokens := bytes.SplitN(tokens[1], []byte(": "), 2)
			if len(keyTokens) != 2 {
				conn.Write([]byte("RCSP/1.0 SET NOT_OK\r\nMESSAGE: Invalid key format\r\n"))
				continue MsgLoop
			}
			val, ok := s.cache.Get(string(keyTokens[1]))
			resp := response{
				command: "SET",
				ok:      ok,
				key:     keyTokens[1],
				value:   val,
			}
			resp.write(conn, resp)
		case "DELETE":

		case "PURGE":

		case "LENGTH":

		case "KEYS":

		case "PING":
			conn.Write([]byte("RCSP/1.0 PING OK\r\n"))
		case "CLOSE":
			conn.Write([]byte("RCSP/1.0 CLOSE OK\r\n"))
			break MsgLoop
		default:
			conn.Write([]byte("RCSP/1.0 NOT_OK\r\nMESSAGE: Malformed request\r\n"))
		}
	}
}

func (s *Server) shuttingDown() bool {
	return s.inShutdown.isSet()
}

// --- Helpers: --- //

type response struct {
	command string
	ok      bool
	message string
	key     []byte
	value   []byte
}

func (r response) write(conn net.Conn, resp response) {
	msg := []byte("RCSP/1.0 ")
	msg = append(msg, []byte(resp.command)...)
	if resp.ok {
		msg = append(msg, []byte(" OK\r\n")...)
	} else {
		msg = append(msg, []byte(" NOT_OK\r\n")...)
	}
	if resp.message != "" {
		msg = append(msg, []byte("MESSAGE: ")...)
		msg = append(msg, []byte(resp.message)...)
		msg = append(msg, []byte("\r\n")...)
	}
	if resp.key != nil {
		msg = append(msg, []byte("KEY: ")...)
		msg = append(msg, resp.key...)
		msg = append(msg, []byte("\r\n")...)
	}
	if resp.value != nil {
		msg = append(msg, []byte("VALUE: ")...)
		msg = append(msg, resp.value...)
		msg = append(msg, []byte("\r\n")...)
	}
	conn.Write(msg)
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
