package server

import (
	"context"

	"github.com/nmezhenskyi/rcs/internal/cache"
	pb "github.com/nmezhenskyi/rcs/internal/genproto"
)

type GRPCServer struct {
	pb.UnimplementedCacheServiceServer // Embed for forward compatibility.

	cache *cache.CacheMap
}

// --- Public API: --- //

func NewGRPCServer() *GRPCServer {
	return &GRPCServer{cache: cache.NewCacheMap()}
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

func (s *GRPCServer) Flush(ctx context.Context, in *pb.FlushRequest) (*pb.FlushReply, error) {
	s.cache.Flush()
	return &pb.FlushReply{Ok: true}, nil
}

func (s *GRPCServer) Length(ctx context.Context, in *pb.LengthRequest) (*pb.LengthReply, error) {
	length := s.cache.Length()
	return &pb.LengthReply{Length: int64(length)}, nil
}

func (s *GRPCServer) Ping(ctx context.Context, in *pb.PingRequest) (*pb.PingReply, error) {
	return &pb.PingReply{Message: "Pong"}, nil
}
