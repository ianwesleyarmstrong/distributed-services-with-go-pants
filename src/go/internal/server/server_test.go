package server

import (
	"context"
	"flag"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	api "github.com/ianwesleyarmstrong/distributed-services-with-go-pants/api/v1"
	api_gen "github.com/ianwesleyarmstrong/distributed-services-with-go-pants/api_gen/v1"
	"github.com/ianwesleyarmstrong/distributed-services-with-go-pants/internal/auth"
	"github.com/ianwesleyarmstrong/distributed-services-with-go-pants/internal/config"
	"github.com/ianwesleyarmstrong/distributed-services-with-go-pants/internal/log"
	"go.opencensus.io/examples/exporter"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

var (
	debug = flag.Bool("debug", false, "enable observability for debugging.")
)

func TestMain(m *testing.M) {
	flag.Parse()

	if *debug {
		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		zap.ReplaceGlobals(logger)
	}
	os.Exit(m.Run())
}

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		rootClient api_gen.LogClient,
		nobodyClient api_gen.LogClient,
		config *Config,
	){
		"produce/consume a message to/from the log succeeds": testProduceConsume,
		"produce/consume stream succeeds":                    testProduceConsumeStream,
		"consume past log boundary fails":                    testConsumePastBoundary,
		"unauthorized fails":                                 testUnauthorized,
	} {
		t.Run(
			scenario, func(t *testing.T) {
				rootClient, nobodyClient, config, teardown := setupTest(t, nil)
				defer teardown()
				fn(t, rootClient, nobodyClient, config)
			},
		)
	}
}

func testProduceConsume(t *testing.T, client, _ api_gen.LogClient, config *Config) {
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

func testProduceConsumeStream(t *testing.T, client, _ api_gen.LogClient, config *Config) {
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

func testConsumePastBoundary(t *testing.T, client, _ api_gen.LogClient, config *Config) {
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

func testUnauthorized(t *testing.T, _, client api_gen.LogClient, config *Config) {
	ctx := context.Background()
	produce, err := client.Produce(ctx,
		&api_gen.ProduceRequest{
			Record: &api_gen.Record{
				Value: []byte("hello world"),
			},
		},
	)

	if produce != nil {
		t.Fatalf("produce response should be nil")
	}

	gotCode, wantCode := status.Code(err), codes.PermissionDenied
	if gotCode != wantCode {
		t.Fatalf("got code %d, want: %d", gotCode, wantCode)
	}

	consume, err := client.Consume(ctx, &api_gen.ConsumeRequest{
		Offset: 0,
	})
	if consume != nil {
		t.Fatalf("consume response should be nil")
	}
	gotCode, wantCode = status.Code(err), codes.PermissionDenied
	if gotCode != wantCode {
		t.Fatalf("got code %d, want: %d", gotCode, wantCode)
	}
}

func setupTest(t *testing.T, fn func(*Config)) (rootClient, nobodyClient api_gen.LogClient, cfg *Config, teardown func()) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	newClient := func(crtPath, keyPath string) (*grpc.ClientConn, api_gen.LogClient, []grpc.DialOption) {
		clientTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
			CertFile: crtPath,
			KeyFile:  keyPath,
			CAFile:   config.CAFile,
			Server:   false,
		})

		require.NoError(t, err)
		clientCreds := credentials.NewTLS(clientTLSConfig)

		opts := []grpc.DialOption{grpc.WithTransportCredentials(clientCreds)}

		conn, err := grpc.Dial(l.Addr().String(), opts...)
		require.NoError(t, err)
		client := api_gen.NewLogClient(conn)

		return conn, client, opts
	}

	var rootConn *grpc.ClientConn
	rootConn, rootClient, _ = newClient(
		config.RootClientCertFile,
		config.RootClientKeyFile,
	)

	var nobodyConn *grpc.ClientConn
	nobodyConn, nobodyClient, _ = newClient(
		config.NobodyClientCertFile,
		config.NobodyClientKeyFile,
	)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		ServerAddress: l.Addr().String(),
		Server:        true,
	})
	require.NoError(t, err)
	serverCreds := credentials.NewTLS(serverTLSConfig)

	dir, err := ioutil.TempDir("", "server-test")
	require.NoError(t, err)

	clog, err := log.NewLog(dir, log.Config{})
	require.NoError(t, err)

	authorizer := auth.New(config.ACLModelFile, config.ACLPolicyFile)

	var telemetryExporter *exporter.LogExporter
	if *debug {
		metricsLogFile, err := ioutil.TempFile("", "metrics-*.log")
		require.NoError(t, err)
		t.Logf("metrics log file: %s", metricsLogFile.Name())

		traceslogFile, err := ioutil.TempFile("", "traces-*.log")
		require.NoError(t, err)
		t.Logf("traces log file: %s", traceslogFile.Name())

		telemetryExporter, err = exporter.NewLogExporter(exporter.Options{
			MetricsLogFile:    metricsLogFile.Name(),
			TracesLogFile:     traceslogFile.Name(),
			ReportingInterval: time.Second,
		})

		require.NoError(t, err)
		err = telemetryExporter.Start()
		require.NoError(t, err)
	}

	cfg = &Config{
		CommitLog:  clog,
		Authorizer: authorizer,
	}

	if fn != nil {
		fn(cfg)
	}

	server, err := NewGRPCServer(cfg, grpc.Creds(serverCreds))
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	return rootClient, nobodyClient, cfg, func() {
		server.Stop()
		nobodyConn.Close()
		rootConn.Close()
		l.Close()
		clog.Remove()
		if telemetryExporter != nil {
			time.Sleep(1500 * time.Millisecond)
			telemetryExporter.Stop()
			telemetryExporter.Close()
		}
	}
}
