//go:build rmhttp

package httpsrv

import (
	"context"

	"github.com/nmezhenskyi/rcs/internal/cache"
	"github.com/rs/zerolog"
)

type Server struct {
	Logger zerolog.Logger
}

func NewServer(_ *cache.CacheMap) *Server {
	return &Server{}
}

func (s *Server) ListenAndServe(addr string) error {
	return nil
}

func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return nil
}

func (s *Server) Close(ctx context.Context) error {
	return nil
}
