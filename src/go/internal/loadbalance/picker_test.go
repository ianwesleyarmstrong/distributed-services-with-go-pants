package loadbalance_test

import (
	"testing"

	// "google.golang.org/grpc/attributes"
	"google.golang.org/grpc/balancer"
	// "google.golang.org/grpc/balancer/base"
	// "google.golang.org/grpc/resolver"

	"github.com/stretchr/testify/require"
	
	"github.com/ianwesleyarmstrong/distributed-services-with-go-pants/internal/loadbalance"
)

func TestPickerNoSubConnAvailable(t *testing.T) {
	picker := &loadbalance.Picker{}
	for _, method := range []string{
		"/log.vX.log/Produce",
		"/log.vX.log/Consume",
	} {
		info := balancer.PickInfo{
			FullMethodName: method,
		}

		result, err := picker.Pick(info)
		require.Equal(t, balancer.ErrNoSubConnAvailable, err)
		require.Nil(t, result.SubConn)
	}
}