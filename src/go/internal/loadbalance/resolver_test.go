package loadbalance_test

import (
	"flag"
	"net"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"

	api_gen "github.com/ianwesleyarmstrong/distributed-services-with-go-pants/api_gen/v1"
	"github.com/ianwesleyarmstrong/distributed-services-with-go-pants/internal/config"
	"github.com/ianwesleyarmstrong/distributed-services-with-go-pants/internal/loadbalance"
	"github.com/ianwesleyarmstrong/distributed-services-with-go-pants/internal/server"
)

var (
	debug = flag.Bool("debug", false, "enable observability for debugging.")
)

func TestMain(m *testing.M) {
	logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		zap.ReplaceGlobals(logger)
	os.Exit(m.Run())
}

func TestResolver(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		Server:        true,
		ServerAddress: "127.0.0.1",
	})
	require.NoError(t, err)
	serverCreds := credentials.NewTLS(tlsConfig)

	srv, err := server.NewGRPCServer(&server.Config{
		GetServerer: &getServers{},
	}, grpc.Creds(serverCreds))
	require.NoError(t, err)

	go srv.Serve(l)

	conn := &mockClientConn{}
	tlsConfig, err = config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.RootClientCertFile,
		KeyFile:       config.RootClientKeyFile,
		CAFile:        config.CAFile,
		Server:        false,
		ServerAddress: "127.0.0.1",
	})
	require.NoError(t, err)
	clientCreds := credentials.NewTLS(tlsConfig)
	opts := resolver.BuildOptions{
		DialCreds: clientCreds,
	}
	r := &loadbalance.Resolver{}
	_, err = r.Build(
		resolver.Target{
			URL: url.URL{Path: l.Addr().String()},
		},
		conn,
		opts,
	)
	require.NoError(t, err)

	wantState := resolver.State{
		Addresses: []resolver.Address{{
			Addr:       "localhost:9001",
			Attributes: attributes.New("is_leader", true),
		}, {
			Addr:       "localhost:9002",
			Attributes: attributes.New("is_leader", false),
		}},
	}
	require.Equal(t, wantState, conn.state)

	conn.state.Addresses = nil
	r.ResolveNow(resolver.ResolveNowOptions{})
	require.Equal(t, wantState, conn.state)
}

type getServers struct{}

func (s *getServers) GetServers() ([]*api_gen.Server, error) {
	return []*api_gen.Server{{
		Id:       "leader",
		RpcAddr:  "localhost:9001",
		IsLeader: true,
	}, {
		Id:      "follower",
		RpcAddr: "localhost:9002",
	}}, nil
}

type mockClientConn struct {
	resolver.ClientConn
	state resolver.State
}

func (c *mockClientConn) UpdateState(state resolver.State) error {
	c.state = state
	return nil
}

func (c *mockClientConn) ReportError(err error) {}

func (c *mockClientConn) NewAddress(addrs []resolver.Address) {}

func (c *mockClientConn) NewServiceConfig(config string) {}

func (c *mockClientConn) ParseServiceConfig(
	config string,
) *serviceconfig.ParseResult {
	return nil
}
