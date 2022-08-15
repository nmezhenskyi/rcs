package nativesrv

import (
	"bytes"
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
		var resp = response{}

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
			resp.command = []byte("SET")
			if len(tokens) != 3 {
				resp.ok = false
				resp.message = []byte("Key and/or value are missing")
				resp.write(conn)
				continue MsgLoop
			}
			keyTokens := bytes.SplitN(tokens[1], []byte(": "), 2)
			valueTokens := bytes.SplitN(tokens[2], []byte(": "), 2)
			if len(keyTokens) != 2 || len(valueTokens) != 2 {
				resp.ok = false
				resp.message = []byte("Invalid key and/or value format")
				resp.write(conn)
				continue MsgLoop
			}
			key := strings.TrimSuffix(string(keyTokens[1]), "\r\n")
			s.cache.Set(key, valueTokens[1])
			resp.ok = true
			resp.key = []byte(key)
			resp.write(conn)
		case "GET":
			resp.command = []byte("GET")
			if len(tokens) != 2 {
				resp.ok = false
				resp.message = []byte("Key is missing")
				resp.write(conn)
				continue MsgLoop
			}
			keyTokens := bytes.SplitN(tokens[1], []byte(": "), 2)
			if len(keyTokens) != 2 {
				resp.ok = false
				resp.message = []byte("Invalid key format")
				resp.write(conn)
				continue MsgLoop
			}
			val, ok := s.cache.Get(strings.TrimSuffix(string(keyTokens[1]), "\r\n"))
			resp.ok = ok
			resp.key = keyTokens[1]
			resp.value = val
			resp.write(conn)
		case "DELETE":

		case "PURGE":

		case "LENGTH":
			length := s.cache.Length()
			resp.command = []byte("LENGTH")
			resp.ok = true
			resp.value = []byte(strconv.Itoa(length))
			resp.write(conn)
		case "KEYS":

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
			conn.Write([]byte("RCSP/1.0 NOT_OK\r\nMESSAGE: Malformed request\r\n"))
		}
	}
}

func (s *Server) shuttingDown() bool {
	return s.inShutdown.isSet()
}

// --- Helpers: --- //

type response struct {
	command []byte
	ok      bool
	message []byte
	key     []byte
	value   []byte
}

func (r response) write(conn net.Conn) {
	msg := []byte("RCSP/1.0")
	if r.command != nil {
		msg = append(msg, ' ')
		msg = append(msg, r.command...)
	}
	if r.ok {
		msg = append(msg, []byte(" OK\r\n")...)
	} else {
		msg = append(msg, []byte(" NOT_OK\r\n")...)
	}
	if r.message != nil {
		msg = append(msg, []byte("MESSAGE: ")...)
		msg = append(msg, r.message...)
		msg = append(msg, []byte("\r\n")...)
	}
	if r.key != nil {
		msg = append(msg, []byte("KEY: ")...)
		msg = append(msg, r.key...)
		msg = append(msg, []byte("\r\n")...)
	}
	if r.value != nil {
		msg = append(msg, []byte("VALUE: ")...)
		msg = append(msg, r.value...)
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
