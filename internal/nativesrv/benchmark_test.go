package nativesrv

import (
	"net"
	"testing"
)

func BenchmarkSet(b *testing.B) {
	server := NewServer(nil)
	serverAddr := "localhost:5000"
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			b.Errorf("Server failed: %v", err)
		}
	}()
	defer server.Close()

	req := request{
		command: []byte("SET"),
		key:     []byte("apollo"),
		value:   []byte("Apollo is one of the Olympian deities in classical Greek and Roman religion and Greek and Roman mythology. (From Wikipedia, the free encyclopedia)"),
	}
	respBuf := [1024]byte{}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		conn, err := net.Dial("tcp", serverAddr)
		if err != nil {
			b.Errorf("Failed to connect to the server: %v", err)
		}
		req.write(conn)
		n, err := conn.Read(respBuf[:])
		if err != nil || n == 0 {
			b.Errorf("Error while reading from server")
		}
	}
}
