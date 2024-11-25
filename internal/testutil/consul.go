package testutil

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/consul"

	"github.com/davidsbond/autopgo/internal/target"
)

func ConsulContainer(t *testing.T) *api.Client {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	container, err := consul.Run(ctx, consul.DefaultBaseImage)
	require.NoError(t, err)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		require.NoError(t, container.Terminate(ctx))
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "8500")
	require.NoError(t, err)

	client, err := api.NewClient(&api.Config{
		Address: net.JoinHostPort(host, port.Port()),
	})
	require.NoError(t, err)

	return client
}

func ConsulTarget(t *testing.T, client *api.Client) target.Target {
	service := &api.AgentServiceRegistration{
		ID:      "test",
		Name:    "test",
		Address: "127.0.0.1",
		Port:    8080,
		Tags: []string{
			"autopgo.scrape=true",
			"autopgo.scrape.app=test",
			"autopgo.scrape.scheme=https",
			"autopgo.scrape.path=/test/app",
		},
	}

	require.NoError(t, client.Agent().ServiceRegister(service))

	return target.Target{
		Address: "https://127.0.0.1:8080",
		Path:    "/test/app",
	}
}
