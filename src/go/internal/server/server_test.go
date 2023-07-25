package server

import (
	"context"
	"io/ioutil"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/ianwesleyarmstrong/distributed-services-with-go-pants/api/v1"
	api_gen "github.com/ianwesleyarmstrong/distributed-services-with-go-pants/api_gen/v1"
	"github.com/ianwesleyarmstrong/distributed-services-with-go-pants/internal/log"
	"google.golang.org/grpc"
)

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		client api_gen.LogClient,
		config *Config,
	){
		"produce/consume a message to/from the log succeeds": testProduceConsume,
		"produce/consume stream succeeds":                    testProduceConsumeStream,
		"consume past log boundary fails":                    testConsumePastBoundary,
	} {
		t.Run(
			scenario, func(t *testing.T) {
				client, config, teardown := setupTest(t, nil)
				defer teardown()
				fn(t, client, config)
			},
		)
	}
}

func testProduceConsume(t *testing.T, client api_gen.LogClient, config *Config) {
	ctx := context.Background()

	want := &api_gen.Record{
		Value: []byte("hello world"),
	}

	produce, err := client.Produce(
		ctx,
		&api_gen.ProduceRequest{
			Record: want,
		},
	)
	require.NoError(t, err)

	consume, err := client.Consume(
		ctx,
		&api_gen.ConsumeRequest{
			Offset: produce.Offset,
		},
	)
	require.NoError(t, err)
	require.Equal(t, want.Value, consume.Record.Value)
	require.Equal(t, want.Offset, consume.Record.Offset)
}

func testProduceConsumeStream(t *testing.T, client api_gen.LogClient, config *Config) {
	ctx := context.Background()

	records := []*api_gen.Record{
		{
			Value:  []byte("first message"),
			Offset: 0,
		},
		{
			Value:  []byte("second message"),
			Offset: 1,
		},
	}

	{
		stream, err := client.ProduceStream(ctx)
		require.NoError(t, err)

		for offset, record := range records {
			err = stream.Send(&api_gen.ProduceRequest{
				Record: record,
			})
			require.NoError(t, err)
			res, err := stream.Recv()
			require.NoError(t, err)
			if res.Offset != uint64(offset) {
				t.Fatalf(
					"got offset: %d, want offset: %d",
					res.Offset,
					offset,
				)
			}
		}
	}

	{
		stream, err := client.ConsumeStream(ctx, &api_gen.ConsumeRequest{Offset: 0})
		require.NoError(t, err)

		for i, record := range records {
			res, err := stream.Recv()
			require.NoError(t, err)
			require.Equal(t, res.Record, &api_gen.Record{
				Value:  record.Value,
				Offset: uint64(i),
			})
		}
	}
}

func testConsumePastBoundary(t *testing.T, client api_gen.LogClient, config *Config) {
	ctx := context.Background()

	produce, err := client.Produce(ctx, &api_gen.ProduceRequest{
		Record: &api_gen.Record{
			Value: []byte("hello world"),
		},
	})
	require.NoError(t, err)

	consume, err := client.Consume(ctx, &api_gen.ConsumeRequest{
		Offset: produce.Offset + 1,
	})

	if consume != nil {
		t.Fatal("consume not nil")
	}

	got := grpc.Code(err)
	want := grpc.Code(api.ErrOffsetOutOfRange{}.GRPCStatus().Err())

	if got != want {
		t.Fatalf("got err: %v, want %v", got, want)
	}
}

func setupTest(t *testing.T, fn func(*Config)) (client api_gen.LogClient, cfg *Config, teardown func()) {
	t.Helper()

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	clientOptions := []grpc.DialOption{grpc.WithInsecure()}
	cc, err := grpc.Dial(l.Addr().String(), clientOptions...)
	require.NoError(t, err)

	dir, err := ioutil.TempDir("", "server-test")
	require.NoError(t, err)

	clog, err := log.NewLog(dir, log.Config{})
	require.NoError(t, err)

	cfg = &Config{
		CommitLog: clog,
	}

	if fn != nil {
		fn(cfg)
	}

	server, err := NewGRPCServer(cfg)
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	client = api_gen.NewLogClient(cc)

	return client, cfg, func() {
		server.Stop()
		cc.Close()
		l.Close()
		clog.Remove()
	}
}
