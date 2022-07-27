package nativesrv

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/nmezhenskyi/rcs/internal/cache"
)

type Server struct {
	cache *cache.CacheMap

	addr       string
	inShutdown atomicBool

	mu          sync.Mutex
	listener    *srvListener
	activeConns map[*net.TCPConn]struct{}
}

// --- Public API: --- //

func NewServer(c *cache.CacheMap) *Server {
	if c == nil {
		c = cache.NewCacheMap()
	}
	return &Server{
		cache:       c,
		activeConns: make(map[*net.TCPConn]struct{}),
	}
}

func (s *Server) ListenAndServe(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.serve(listener)
}

func (s *Server) Shutdown() {

}

func (s *Server) Close() {
	s.inShutdown.setTrue()
	s.mu.Lock()
	defer s.mu.Unlock()
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
			return err
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {

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
