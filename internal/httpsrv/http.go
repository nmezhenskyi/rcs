//go:build !rmhttp

// Package httpsrv implements HTTP server that uses JSON payloads for data communication.
//
// See OpenAPI specification at https://github.com/nmezhenskyi/rcs/blob/main/api/openapi/rcs.yaml.
package httpsrv

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/nmezhenskyi/rcs/internal/cache"
	"github.com/rs/zerolog"
)

// Server implements RCS HTTP API according to specification.
type Server struct {
	server *http.Server
	router *httprouter.Router
	cache  *cache.CacheMap

	Logger zerolog.Logger // By defaut Logger is disabled, but can be manually attached.
}

// NewServer initializes a new Server instance ready to be used and returns a pointer to it.
// A zerolog.Logger can be attached to returned Server by accessing public field Server.Logger.
func NewServer(c *cache.CacheMap) *Server {
	if c == nil {
		c = cache.NewCacheMap()
	}
	s := &Server{
		router: httprouter.New(),
		server: &http.Server{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
			TLSConfig: &tls.Config{
				CurvePreferences: []tls.CurveID{
					tls.CurveP256,
					tls.X25519,
				},
			},
		},
		cache:  c,
		Logger: zerolog.New(os.Stderr).Level(zerolog.Disabled),
	}
	s.server.Handler = s.router
	s.setupRoutes()
	return s
}

// ServeHTTP makes the server implement the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// ListenAndServe listens on the given TCP network address addr and
// handles requests on incoming connections according to RCS HTTP API specification.
//
// Unlike http.Server, it does not return ErrServerClosed after Shutdown or Close.
func (s *Server) ListenAndServe(addr string) error {
	s.server.Addr = addr
	s.Logger.Info().Msg("Starting http server on " + addr)
	err := s.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		s.Logger.Error().Err(err).Msg("http server failed")
	}
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// ListenAndServeTLS listens on the given TCP network address addr and
// handles requests on incoming TLS connections according to RCS HTTP API specification.
//
// Unlike http.Server, it does not return ErrServerClosed after Shutdown or Close.
func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	s.server.Addr = addr
	s.Logger.Info().Msg("Starting tls http server on " + addr)
	err := s.server.ListenAndServeTLS(certFile, keyFile)
	if err != nil && err != http.ErrServerClosed {
		s.Logger.Error().Err(err).Msg("http server failed")
	}
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Shutdown gracefully shuts down the server without interrupting any
// active connections. Waits until all connections are closed or until context
// timeout runs out. Once Shutdown has been called on a server, it may not be reused.
//
// Shutdown returns an error returned from closing the Server's underlying listener or
// a context error.
func (s *Server) Shutdown(ctx context.Context) error {
	err := s.server.Shutdown(ctx)
	if err != nil {
		s.Logger.Error().Err(err).Msg("http server shutdown failed")
	} else {
		s.Logger.Info().Msg("http server has been shutdown")
	}
	return err
}

// Close immediately closes all active connections and underlying listener.
// For a graceful shutdown, use Shutdown. Once Close has been called on a server,
// it may not be reused.
//
// Close returns any error returned from closing the Server's underlying listener.
func (s *Server) Close() error {
	err := s.server.Close()
	if err != nil {
		s.Logger.Error().Err(err).Msg("http server has been closed & returned error")
	} else {
		s.Logger.Info().Msg("http server has been closed")
	}
	return err
}

func (s *Server) setupRoutes() {
	s.router.PUT("/SET/:key", s.handleSet())
	s.router.GET("/GET/:key", s.handleGet())
	s.router.DELETE("/DELETE/:key", s.handleDelete())
	s.router.DELETE("/PURGE", s.handlePurge())
	s.router.GET("/LENGTH", s.handleLength())
	s.router.GET("/KEYS", s.handleKeys())
	s.router.GET("/PING", s.handlePing())
}

func (s *Server) handleSet() httprouter.Handle {
	type request struct {
		Value string `json:"value"`
	}
	return func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		s.Logger.Debug().Msg("received http PUT \"/SET/:key\" request from " + req.RemoteAddr)

		key := p.ByName("key")
		if key == "" {
			sendBadRequest(w, "SET", "Key cannot be empty")
			return
		}
		reqData := request{}
		err := json.NewDecoder(req.Body).Decode(&reqData)
		if err != nil {
			sendBadRequest(w, "SET", "Failed to decode request body")
			return
		}
		if len(reqData.Value) == 0 {
			sendBadRequest(w, "SET", "Value cannot be empty")
			return
		}

		s.cache.Set(key, []byte(reqData.Value))

		res := httpResponse{
			Command: "SET",
			Key:     key,
			Ok:      true,
		}
		sendJSON(w, 200, res)
	}
}

func (s *Server) handleGet() httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		s.Logger.Debug().Msg("received http GET \"/GET/:key\" request from " + req.RemoteAddr)

		key := p.ByName("key")
		if key == "" {
			sendBadRequest(w, "GET", "Key cannot be empty")
			return
		}

		value, ok := s.cache.Get(key)

		res := httpResponse{
			Command: "GET",
			Key:     key,
			Value:   string(value),
			Ok:      ok,
		}
		sendJSON(w, 200, res)
	}
}

func (s *Server) handleDelete() httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		s.Logger.Debug().Msg("received http DELETE \"/DELETE/:key\" request from " + req.RemoteAddr)

		key := p.ByName("key")
		if key == "" {
			sendBadRequest(w, "DELETE", "Key cannot be empty")
			return
		}

		s.cache.Delete(key)

		res := httpResponse{
			Command: "DELETE",
			Key:     key,
			Ok:      true,
		}
		sendJSON(w, 200, res)
	}
}

func (s *Server) handlePurge() httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		s.Logger.Debug().Msg("received http DELETE \"/PURGE\" request from " + req.RemoteAddr)
		s.cache.Purge()
		res := httpResponse{
			Command: "FLUSH",
			Ok:      true,
		}
		sendJSON(w, 200, res)
	}
}

func (s *Server) handleLength() httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		s.Logger.Debug().Msg("received http GET \"/LENGTH\" request from " + req.RemoteAddr)
		length := s.cache.Length()
		res := httpResponse{
			Command: "LENGTH",
			Value:   length,
			Ok:      true,
		}
		sendJSON(w, 200, res)
	}
}

func (s *Server) handleKeys() httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		s.Logger.Debug().Msg("received http GET \"/KEYS\" request from " + req.RemoteAddr)
		keys := s.cache.Keys()
		res := httpResponse{
			Command: "KEYS",
			Value:   keys,
			Ok:      true,
		}
		sendJSON(w, 200, res)
	}
}

func (s *Server) handlePing() httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		s.Logger.Debug().Msg("received http GET \"/PING\" request from " + req.RemoteAddr)
		sendJSON(w, 200, httpResponse{Command: "PING", Message: "PONG", Ok: true})
	}
}

type httpResponse struct {
	Command string `json:"command"`
	Message string `json:"message,omitempty"`
	Key     string `json:"key,omitempty"`
	Value   any    `json:"value,omitempty"`
	Ok      bool   `json:"ok"`
}

func sendJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(statusCode)
	if body != nil {
		json.NewEncoder(w).Encode(body)
	} else {
		json.NewEncoder(w).Encode(struct{}{})
	}
}

func sendBadRequest(w http.ResponseWriter, command, message string) {
	res := httpResponse{
		Command: command,
		Message: message,
		Ok:      false,
	}
	sendJSON(w, 400, res)
}
