package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/nmezhenskyi/rcs/internal/cache"
)

type HTTPServer struct {
	server *http.Server
	router *httprouter.Router
	cache  *cache.CacheMap
}

// --- Public API: --- //

func NewHTTPServer() *HTTPServer {
	s := &HTTPServer{
		router: httprouter.New(),
		server: &http.Server{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
			TLSConfig: &tls.Config{
				PreferServerCipherSuites: true,
				CurvePreferences: []tls.CurveID{
					tls.CurveP256,
					tls.X25519,
				},
			},
		},
		cache: cache.NewCacheMap(),
	}
	s.server.Handler = s.router
	s.setupRoutes()
	return s
}

func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *HTTPServer) ListenAndServe(addr string) error {
	s.server.Addr = addr
	return s.server.ListenAndServe()
}

func (s *HTTPServer) ListenAndServeTLS(addr, certFile, keyFile string) error {
	s.server.Addr = addr
	return s.server.ListenAndServeTLS(certFile, keyFile)
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *HTTPServer) Close() error {
	return s.server.Close()
}

// --- Private: --- //

func (s *HTTPServer) setupRoutes() {
	s.router.PUT("/SET/:key", s.handleSet())
	s.router.GET("/GET/:key", s.handleGet())
	s.router.DELETE("/DELETE/:key", s.handleDelete())
	s.router.DELETE("/PURGE", s.handlePurge())
	s.router.GET("/LENGTH", s.handleLength())
	s.router.GET("/PING", s.handlePing())
}

func (s *HTTPServer) handleSet() httprouter.Handle {
	type request struct {
		Value string `json:"value"`
	}
	return func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		key := p.ByName("key")
		if key == "" {
			sendBadRequest(w, "SET", "Key cannot be an empty string")
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

func (s *HTTPServer) handleGet() httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, p httprouter.Params) {
		key := p.ByName("key")
		if key == "" {
			sendBadRequest(w, "GET", "Key cannot be an empty string")
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

func (s *HTTPServer) handleDelete() httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, p httprouter.Params) {
		key := p.ByName("key")
		if key == "" {
			sendBadRequest(w, "DELETE", "Key cannot be an empty string")
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

func (s *HTTPServer) handlePurge() httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		s.cache.Purge()
		res := httpResponse{
			Command: "FLUSH",
			Ok:      true,
		}
		sendJSON(w, 200, res)
	}
}

func (s *HTTPServer) handleLength() httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		length := s.cache.Length()
		res := httpResponse{
			Command: "LENGTH",
			Value:   length,
			Ok:      true,
		}
		sendJSON(w, 200, res)
	}
}

func (s *HTTPServer) handlePing() httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		sendJSON(w, 200, httpResponse{Command: "PING", Message: "PONG", Ok: true})
	}
}

// --- Helpers: --- //

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
