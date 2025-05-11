package server

import (
	"context"
	"github.com/drtcrz23/project_go/internal/config"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/drtcrz23/project_go/internal/subpub"
	pb "github.com/drtcrz23/project_go/proto"
)

type Server struct {
	pb.UnimplementedPubSubServer
	config    *config.Config
	subPub    subpub.SubPub
	logger    *logrus.Logger
	mu        sync.RWMutex
	closed    bool
	closeChan chan struct{}
}

func NewServer(config *config.Config, subPub subpub.SubPub, logger *logrus.Logger) *Server {
	return &Server{
		config:    config,
		subPub:    subPub,
		logger:    logger,
		closeChan: make(chan struct{}),
	}
}

func (s *Server) Subscribe(req *pb.SubscribeRequest, stream pb.PubSub_SubscribeServer) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return status.Error(codes.Unavailable, "server is closed")
	}
	s.mu.RUnlock()

	s.logger.WithField("key", req.Key).Info("New subscription")

	handler := func(msg interface{}) {
		data, ok := msg.(string)
		if !ok {
			s.logger.WithField("key", req.Key).Error("Invalid message type")
			return
		}
		if err := stream.Send(&pb.Event{Data: data}); err != nil {
			s.logger.WithField("key", req.Key).Errorf("Failed to send event: %v", err)
		}
	}

	sub, err := s.subPub.Subscribe(req.Key, handler)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	<-stream.Context().Done()
	return nil
}

func (s *Server) Publish(ctx context.Context, req *pb.PublishRequest) (*emptypb.Empty, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, status.Error(codes.Unavailable, "server is closed")
	}
	s.mu.RUnlock()

	s.logger.WithField("key", req.Key).Info("Publishing message")

	if err := s.subPub.Publish(req.Key, req.Data); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to publish: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.config.GRPCPort)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPubSubServer(grpcServer, s)

	go func() {
		s.logger.Infof("Starting gRPC server on %s", s.config.GRPCPort)
		if err := grpcServer.Serve(listener); err != nil {
			s.logger.Errorf("gRPC server error: %v", err)
		}
	}()

	// Handle graceful shutdown
	select {
	case <-ctx.Done():
		s.logger.Info("Context cancelled")
	}

	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()

	grpcServer.GracefulStop()
	return s.subPub.Close(ctx)
}
