// Package grpcsrv implements gRPC server.
//
// See Protobuf at https://github.com/nmezhenskyi/rcs/blob/main/api/protobuf/rcs.proto.
package grpcsrv

import (
	"context"
	"crypto/tls"
	"net"
	"os"
	"time"

	"github.com/nmezhenskyi/rcs/internal/cache"
	pb "github.com/nmezhenskyi/rcs/internal/genproto"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Server implements RCS gRPC service.
type Server struct {
	pb.UnimplementedCacheServiceServer // Embed for forward compatibility.

	server *grpc.Server
	cache  *cache.CacheMap
	opts   []grpc.ServerOption

	Logger zerolog.Logger // By defaut Logger is disabled, but can be manually attached.
}

func NewServer(c *cache.CacheMap, opts ...grpc.ServerOption) *Server {
	if c == nil {
		c = cache.NewCacheMap()
	}
	srv := &Server{
		server: nil, // Will be initialized in ListenAndServe / ListenAndServeTLS
		cache:  c,
		opts:   opts,
		Logger: zerolog.New(os.Stderr).Level(zerolog.Disabled),
	}
	return srv
}

func (s *Server) Set(ctx context.Context, in *pb.SetRequest) (*pb.SetReply, error) {
	key := in.GetKey()
	value := in.GetValue()
	if len(key) == 0 {
		return &pb.SetReply{Key: key, Ok: false, Message: "Key cannot be empty"}, nil
	}
	if len(value) == 0 {
		return &pb.SetReply{Key: key, Ok: false, Message: "Value cannot be empty"}, nil
	}
	s.cache.Set(key, value)
	return &pb.SetReply{Key: key, Ok: true}, nil
}

func (s *Server) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetReply, error) {
	key := in.GetKey()
	if len(key) == 0 {
		return &pb.GetReply{Key: key, Ok: false, Message: "Key cannot be empty"}, nil
	}
	value, ok := s.cache.Get(key)
	if !ok {
		return &pb.GetReply{Key: key, Value: value, Ok: ok, Message: "Value not found"}, nil
	}
	return &pb.GetReply{Key: key, Value: value, Ok: ok}, nil
}

func (s *Server) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteReply, error) {
	key := in.GetKey()
	if len(key) == 0 {
		return &pb.DeleteReply{Key: key, Ok: false, Message: "Key cannot be empty"}, nil
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
	s.Logger.Info().Msg("Starting grpc server on " + addr)
	s.server = grpc.NewServer(s.opts...)
	pb.RegisterCacheServiceServer(s.server, s)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		s.Logger.Error().Err(err).Msg("failed to start listener")
		return err
	}
	return s.server.Serve(lis)
}

// ListenAndServeTLS
func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		s.Logger.Error().Err(err).Msg("failed to load tls certificate")
		return err
	}
	creds := credentials.NewServerTLSFromCert(&cert)
	s.opts = append(s.opts, grpc.Creds(creds))
	s.server = grpc.NewServer(s.opts...)
	pb.RegisterCacheServiceServer(s.server, s)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.Logger.Error().Err(err).Msg("failed to start tls listener")
		return err
	}
	return s.server.Serve(listener)
}

// Shutdown gracefully shuts down the server without interrupting any
// active connections. Accepts context with timeout that will forcefully close
// the server if timeout runs out.
func (s *Server) Shutdown(ctx context.Context) error {
	stopped := make(chan struct{}, 1)
	go func() {
		s.server.GracefulStop()
		close(stopped)
		s.Logger.Info().Msg("grpc server has been shutdown")
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-stopped:
			return nil
		default:
			time.Sleep(50 * time.Millisecond)
			continue
		}
	}
}

// Close immediately closes all active connections and listeners.
// For a graceful shutdown, use Shutdown.
func (s *Server) Close() {
	s.server.Stop()
	s.Logger.Info().Msg("grpc server has been closed")
}
