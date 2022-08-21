package grpcsrv

import (
	"bytes"
	"context"
	"testing"

	pb "github.com/nmezhenskyi/rcs/internal/genproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestNewServer(t *testing.T) {
	server := NewServer(nil)
	if server == nil {
		t.Error("Expected pointer to initialized Server, got nil instead")
	}
	if server.server == nil {
		t.Error("Server.server has not been initialized")
	}
	if server.cache == nil {
		t.Error("Server.cache has not been initialized")
	}
}

func TestSet(t *testing.T) {
	server := NewServer(nil)
	serverAddr := "localhost:5001"
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()
	client, conn := newTestClient(serverAddr, t)
	defer conn.Close()
	defer server.Close()

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
}

func TestGet(t *testing.T) {
	server := NewServer(nil)
	serverAddr := "localhost:5001"
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()
	client, conn := newTestClient(serverAddr, t)
	defer conn.Close()
	defer server.Close()

	testCases := []struct {
		name  string
		key   string
		value []byte
		ok    bool
	}{
		{
			name:  "Valid key, valid value",
			key:   "key1",
			value: []byte("10"),
			ok:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.key != "" {
				server.cache.Set(tc.key, tc.value)
			}
			reqData := &pb.GetRequest{Key: tc.key}
			reply, err := client.Get(context.Background(), reqData)
			if err != nil {
				t.Errorf("Failed to send the request: %v", err)
			}
			if reply.Ok != tc.ok {
				t.Errorf("Expected Ok to be %t, got %t instead", tc.ok, reply.Ok)
			}
			if reply.Key != tc.key {
				t.Errorf("Expected key \"%s\", got \"%s\" instead", tc.key, reply.Key)
			}
			if bytes.Compare(reply.Value, tc.value) != 0 {
				t.Errorf("Expected value to be %v, got %v instead", tc.value, reply.Value)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	server := NewServer(nil)
	serverAddr := "localhost:5001"
	server.cache.Set("key1", []byte("10"))
	server.cache.Set("key2", []byte("20"))
	server.cache.Set("key3", []byte("30"))
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()
	client, conn := newTestClient(serverAddr, t)
	defer conn.Close()
	defer server.Close()

	lengthBefore := server.cache.Length()
	reqData := &pb.DeleteRequest{Key: "key1"}
	reply, err := client.Delete(context.Background(), reqData)
	if err != nil {
		t.Errorf("Failed to send the request: %v", err)
	}
	if !reply.Ok {
		t.Errorf("Expected Ok to be true, got %t instead", reply.Ok)
	}
	if server.cache.Length() != lengthBefore-1 {
		t.Errorf("Cache length has not changed")
	}
}

func TestPurge(t *testing.T) {
	server := NewServer(nil)
	serverAddr := "localhost:5001"
	server.cache.Set("key1", []byte("10"))
	server.cache.Set("key2", []byte("20"))
	server.cache.Set("key3", []byte("30"))
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()
	client, conn := newTestClient(serverAddr, t)
	defer conn.Close()
	defer server.Close()

	reqData := &pb.PurgeRequest{}
	reply, err := client.Purge(context.Background(), reqData)
	if err != nil {
		t.Errorf("Failed to send the request: %v", err)
	}
	if !reply.Ok {
		t.Errorf("Expected Ok to be true, got %t instead", reply.Ok)
	}
	if server.cache.Length() != 0 {
		t.Errorf("Cache is not empty")
	}
}

func TestLength(t *testing.T) {
	server := NewServer(nil)
	serverAddr := "localhost:5001"
	server.cache.Set("key1", []byte("10"))
	server.cache.Set("key2", []byte("20"))
	server.cache.Set("key3", []byte("30"))
	server.cache.Set("key4", []byte("40"))
	server.cache.Set("key5", []byte("50"))
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()
	client, conn := newTestClient(serverAddr, t)
	defer conn.Close()
	defer server.Close()

	actualLength := server.cache.Length()
	reqData := &pb.LengthRequest{}
	reply, err := client.Length(context.Background(), reqData)
	if err != nil {
		t.Errorf("Failed to send the request: %v", err)
	}
	if !reply.Ok {
		t.Errorf("Expected Ok to be true, got %t instead", reply.Ok)
	}
	if reply.Length != int64(actualLength) {
		t.Errorf("Expected length %d, got %d instead", actualLength, reply.Length)
	}
}

func TestPing(t *testing.T) {
	server := NewServer(nil)
	serverAddr := "localhost:5001"
	server.cache.Set("key1", []byte("10"))
	server.cache.Set("key2", []byte("20"))
	server.cache.Set("key3", []byte("30"))
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()
	client, conn := newTestClient(serverAddr, t)
	defer conn.Close()
	defer server.Close()

	reqData := &pb.PingRequest{}
	reply, err := client.Ping(context.Background(), reqData)
	if err != nil {
		t.Errorf("Failed to send the request: %v", err)
	}
	if !reply.Ok {
		t.Errorf("Expected Ok to be true, got %t instead", reply.Ok)
	}
	if reply.Message != "PONG" {
		t.Errorf("Expected Message to PONG, got %s instead", reply.Message)
	}
}

func newTestClient(serverAddr string, t *testing.T) (pb.CacheServiceClient, *grpc.ClientConn) {
	var opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		t.Errorf("Failed to connect to the server: %v", err)
	}
	client := pb.NewCacheServiceClient(conn)
	return client, conn
}
