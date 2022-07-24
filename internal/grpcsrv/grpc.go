package grpcsrv

import (
	"context"
	"net"

	"github.com/nmezhenskyi/rcs/internal/cache"
	pb "github.com/nmezhenskyi/rcs/internal/genproto"
	"google.golang.org/grpc"
)

type Server struct {
	pb.UnimplementedCacheServiceServer // Embed for forward compatibility.

	server *grpc.Server
	cache  *cache.CacheMap
}

// --- Public API: --- //

func NewServer(opts ...grpc.ServerOption) *Server {
	s := grpc.NewServer(opts...)
	grpcServer := &Server{
		server: s,
		cache:  cache.NewCacheMap(),
	}
	pb.RegisterCacheServiceServer(s, grpcServer)
	return grpcServer
}

func (s *Server) Set(ctx context.Context, in *pb.SetRequest) (*pb.SetReply, error) {
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

func (s *Server) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetReply, error) {
	key := in.GetKey()
	if len(key) == 0 {
		return &pb.GetReply{Key: key, Ok: false}, nil
	}
	value, ok := s.cache.Get(key)
	return &pb.GetReply{Key: key, Value: value, Ok: ok}, nil
}

func (s *Server) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteReply, error) {
	key := in.GetKey()
	if len(key) == 0 {
		return &pb.DeleteReply{Key: key, Ok: false}, nil
	}
	s.cache.Delete(key)
	return &pb.DeleteReply{Key: key, Ok: true}, nil
}

func (s *Server) Purge(ctx context.Context, in *pb.PurgeRequest) (*pb.PurgeReply, error) {
	s.cache.Purge()
	return &pb.PurgeReply{Ok: true}, nil
}

func (s *Server) Length(ctx context.Context, in *pb.LengthRequest) (*pb.LengthReply, error) {
	length := s.cache.Length()
	return &pb.LengthReply{Length: int64(length), Ok: true}, nil
}

func (s *Server) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingReply, error) {
	return &pb.PingReply{Message: "PONG", Ok: true}, nil
}

func (s *Server) ListenAndServe(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.server.Serve(lis)
}

func (s *Server) Shutdown() {
	s.server.GracefulStop()
}

func (s *Server) Close() {
	s.server.Stop()
}
