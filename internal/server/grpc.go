package server

import (
	"context"
	"net"

	"github.com/nmezhenskyi/rcs/internal/cache"
	pb "github.com/nmezhenskyi/rcs/internal/genproto"
	"google.golang.org/grpc"
)

type GRPCServer struct {
	pb.UnimplementedCacheServiceServer // Embed for forward compatibility.

	server *grpc.Server
	cache  *cache.CacheMap
}

// --- Public API: --- //

func NewGRPCServer(opts ...grpc.ServerOption) *GRPCServer {
	s := grpc.NewServer(opts...)
	grpcServer := &GRPCServer{
		server: s,
		cache:  cache.NewCacheMap(),
	}
	pb.RegisterCacheServiceServer(s, grpcServer)
	return grpcServer
}

func (s *GRPCServer) Set(ctx context.Context, in *pb.SetRequest) (*pb.SetReply, error) {
	key := in.GetKey()
	value := in.GetValue()
	if len(key) == 0 {
		return &pb.SetReply{Key: key, Ok: false}, nil
	}
	if len(value) == 0 {
		return &pb.SetReply{Key: key, Ok: false}, nil
	}
	s.cache.Set(key, value)
	return &pb.SetReply{Key: key, Ok: true}, nil
}

func (s *GRPCServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetReply, error) {
	key := in.GetKey()
	if len(key) == 0 {
		return &pb.GetReply{Key: key, Ok: false}, nil
	}
	value, ok := s.cache.Get(key)
	return &pb.GetReply{Key: key, Value: value, Ok: ok}, nil
}

func (s *GRPCServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteReply, error) {
	key := in.GetKey()
	if len(key) == 0 {
		return &pb.DeleteReply{Key: key, Ok: false}, nil
	}
	s.cache.Delete(key)
	return &pb.DeleteReply{Key: key, Ok: true}, nil
}

func (s *GRPCServer) Purge(ctx context.Context, in *pb.PurgeRequest) (*pb.PurgeReply, error) {
	s.cache.Purge()
	return &pb.PurgeReply{Ok: true}, nil
}

func (s *GRPCServer) Length(ctx context.Context, in *pb.LengthRequest) (*pb.LengthReply, error) {
	length := s.cache.Length()
	return &pb.LengthReply{Length: int64(length)}, nil
}

func (s *GRPCServer) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingReply, error) {
	return &pb.PingReply{Message: "Pong"}, nil
}

func (s *GRPCServer) ListenAndServe(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.server.Serve(lis)
}

func (s *GRPCServer) Shutdown() {
	s.server.GracefulStop()
}

func (s *GRPCServer) Close() {
	s.server.Stop()
}
