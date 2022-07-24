package grpcsrv

import (
	"context"
	"testing"

	pb "github.com/nmezhenskyi/rcs/internal/genproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestNewServer(t *testing.T) {
	server := NewServer()
	if server == nil {
		t.Error("Expected pointer to initialized GRPCServer, got nil instead")
	}
	if server.server == nil {
		t.Error("GRPCServer.server has not been initialized")
	}
	if server.cache == nil {
		t.Error("GRPCServer.cache has not been initialized")
	}
}

func TestSet(t *testing.T) {
	server := NewServer()
	serverAddr := "localhost:5001"
	go func() {
		server.ListenAndServe(serverAddr)
	}()
	client, conn := newTestClient(serverAddr, t)
	defer conn.Close()

	testCases := []struct {
		name  string
		key   string
		value []byte
		ok    bool
	}{
		{
			name:  "Empty key, no value",
			key:   "",
			value: nil,
			ok:    false,
		},
		{
			name:  "Valid key, no value",
			key:   "key1",
			value: nil,
			ok:    false,
		},
		{
			name:  "Valid key, valid value",
			key:   "key1",
			value: []byte("10"),
			ok:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqData := &pb.SetRequest{Key: tc.key, Value: tc.value}
			reply, err := client.Set(context.Background(), reqData)
			if err != nil {
				t.Errorf("Failed to send the request: %v", err)
			}
			if reply.Ok != tc.ok {
				t.Errorf("Expected Ok to be %t, got %t instead", tc.ok, reply.Ok)
			}
			if reply.Key != tc.key {
				t.Errorf("Expected key \"%s\", got \"%s\" instead", tc.key, reply.Key)
			}
		})
	}

	server.Shutdown()
}

func TestGet(t *testing.T) {

}

func TestDelete(t *testing.T) {

}

func TestPurge(t *testing.T) {

}

func TestLength(t *testing.T) {

}

func TestPing(t *testing.T) {

}

func newTestClient(serverAddr string, t *testing.T) (pb.CacheServiceClient, *grpc.ClientConn) {
	var opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		t.Errorf("Failed to connect to the GRPC server: %v", err)
	}
	client := pb.NewCacheServiceClient(conn)
	return client, conn
}
