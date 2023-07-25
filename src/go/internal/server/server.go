package server

import (
	"context"

	api "github.com/ianwesleyarmstrong/distributed-services-with-go-pants/api/v1"

	api_gen "github.com/ianwesleyarmstrong/distributed-services-with-go-pants/api_gen/v1/log_v1"

	"google.golang.org/grpc"
)

type CommitLog interface {
	Append(*api_gen.Record) (uint64, error)
	Read(uint64) (*api_gen.Record, error)
}

type Config struct {
	CommitLog CommitLog
}

var _ api_gen.LogServer = (*grpcServer)(nil)

func NewGRPCServer(config *Config) (*grpc.Server, error) {
	gsrv := grpc.NewServer()
	srv, err := newgrpcServer(config)
	if err != nil {
		return nil, err
	}
	api_gen.RegisterLogServer(gsrv, srv)
	return gsrv, nil
}

type grpcServer struct {
	api_gen.UnimplementedLogServer
	*Config
}

func newgrpcServer(config *Config) (srv *grpcServer, err error) {
	srv = &grpcServer{
		Config: config,
	}

	return srv, nil
}

func (s *grpcServer) Produce(ctx context.Context, req *api_gen.ProduceRequest) (*api_gen.ProduceResponse, error) {
	offset, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}

	return &api_gen.ProduceResponse{Offset: offset}, nil
}

func (s *grpcServer) Consume(ctx context.Context, req *api_gen.ConsumeRequest) (*api_gen.ConsumeResponse, error) {
	record, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, err
	}
	return &api_gen.ConsumeResponse{Record: record}, nil
}

func (s *grpcServer) ProduceStream(stream api_gen.Log_ProduceStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		res, err := s.Produce(stream.Context(), req)
		if err != nil {
			return err
		}
		if err = stream.Send(res); err != nil {
			return err
		}
	}
}

func (s *grpcServer) ConsumeStream(req *api_gen.ConsumeRequest, stream api_gen.Log_ConsumeStreamServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			res, err := s.Consume(stream.Context(), req)
			switch err.(type) {
			case nil:
			case api.ErrOffsetOutOfRange:
				continue
			default:
				return err
			}

			if err = stream.Send(res); err != nil {
				return err
			}
			req.Offset++
		}
	}
}
