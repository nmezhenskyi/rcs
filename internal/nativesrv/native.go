package nativesrv

import "net"

type Server struct{}

// --- Public API: --- //

func (s *Server) ListenAndServe(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.serve(lis)
}

func (s *Server) Shutdown() {

}

func (s *Server) Close() {

}

// --- Private: --- //

func (s *Server) serve(lis net.Listener) error {
	return nil
}
