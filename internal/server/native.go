package server

import "net"

type NativeServer struct{}

// --- Public API: --- //

func (s *NativeServer) ListenAndServe(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.serve(lis)
}

func (s *NativeServer) Shutdown() {

}

func (s *NativeServer) Close() {

}

// --- Private: --- //

func (s *NativeServer) serve(lis net.Listener) error {
	return nil
}
