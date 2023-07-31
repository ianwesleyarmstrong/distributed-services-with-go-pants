package server

import (
	"context"

	api "github.com/ianwesleyarmstrong/distributed-services-with-go-pants/api/v1"
	api_gen "github.com/ianwesleyarmstrong/distributed-services-with-go-pants/api_gen/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
)

const (
	objectWildcard = "*"
	produceAction  = "produce"
	consumeAction  = "consume"
)

type CommitLog interface {
	Append(*api_gen.Record) (uint64, error)
	Read(uint64) (*api_gen.Record, error)
}

type Authorizer interface {
	Authorize(subject, object, action string) error
}

type Config struct {
	CommitLog  CommitLog
	Authorizer Authorizer
}

type grpcServer struct {
	api_gen.UnimplementedLogServer
	*Config
}

type subjectContextKey struct{}

func subject(ctx context.Context) string {
	return ctx.Value(subjectContextKey{}).(string)
}

var _ api_gen.LogServer = (*grpcServer)(nil)

func NewGRPCServer(config *Config, opts ...grpc.ServerOption) (*grpc.Server, error) {
	opts = append(opts,
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(grpc_auth.StreamServerInterceptor(authenticate)),
		),
		grpc.ChainUnaryInterceptor(
			grpc_middleware.ChainUnaryServer(grpc_auth.UnaryServerInterceptor(authenticate)),
		),
	)
	gsrv := grpc.NewServer(opts...)
	srv, err := newgrpcServer(config)
	if err != nil {
		return nil, err
	}
	api_gen.RegisterLogServer(gsrv, srv)
	return gsrv, nil
}

func newgrpcServer(config *Config) (srv *grpcServer, err error) {
	srv = &grpcServer{
		Config: config,
	}

	return srv, nil
}

func authenticate(ctx context.Context) (context.Context, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.New(
			codes.Unknown,
			"couldn't find peer info",
		).Err()
	}

	if peer.AuthInfo == nil {
		return context.WithValue(ctx, subjectContextKey{}, ""), nil
	}

	tlsInfo := peer.AuthInfo.(credentials.TLSInfo)
	subject := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	ctx = context.WithValue(ctx, subjectContextKey{}, subject)

	return ctx, nil
}

func (s *grpcServer) Produce(ctx context.Context, req *api_gen.ProduceRequest) (*api_gen.ProduceResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		produceAction,
	); err != nil {
		return nil, err
	}

	offset, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}

	return &api_gen.ProduceResponse{Offset: offset}, nil
}

func (s *grpcServer) Consume(ctx context.Context, req *api_gen.ConsumeRequest) (*api_gen.ConsumeResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		produceAction,
	); err != nil {
		return nil, err
	}

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
