package nativesrv

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/nmezhenskyi/rcs/internal/cache"
)

type Server struct {
	addr  string
	cache *cache.CacheMap

	inShutdown atomicBool

	mu sync.Mutex
}

// --- Public API: --- //

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

}

// --- Private: --- //

func (s *Server) serve(listener net.Listener) error {
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {

}

// --- Helpers: --- //

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
