package target

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"

	"github.com/hashicorp/consul/api"

	"github.com/davidsbond/autopgo/internal/logger"
)

type (
	// The ConsulSource type is used to source scrapable targets from a HashiCorp Consul instance.
	ConsulSource struct {
		client *api.Client
		filter string
	}
)

// NewConsulSource returns a new instance of the ConsulSource type that will source targets using the provided Consul
// client. It will search for services tagged with the provided app name.
func NewConsulSource(client *api.Client, app string) *ConsulSource {
	return &ConsulSource{
		client: client,
		filter: fmt.Sprintf(`ServiceTags contains "autopgo.scrape=true" and ServiceTags contains "autopgo.scrape.app=%s"`, app),
	}
}

// List all targets within the Consul catalogue matching the application. This method will use the service catalogue to
// find services that have two main tags: autopgo.scrape=true and autopgo.scrape.app=app. The latter tag should use
// the configured application name as the tag value. A custom path & scheme can be set using the autopgo.scrape.path
// and autopgo.scrape.scheme tags.
func (cs *ConsulSource) List(ctx context.Context) ([]Target, error) {
	log := logger.FromContext(ctx)

	options := &api.QueryOptions{
		Filter: cs.filter,
	}

	log.DebugContext(ctx, "listing consul services")
	serviceNames, _, err := cs.client.Catalog().Services(options.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	log.
		With(slog.Int("count", len(serviceNames))).
		DebugContext(ctx, "found tagged services")

	targets := make([]Target, 0)
	for name := range serviceNames {
		services, _, err := cs.client.Catalog().Service(name, "", options.WithContext(ctx))
		if err != nil {
			return nil, err
		}

		for _, service := range services {
			tags := tagsToMap(service.ServiceTags)

			scheme := tags[schemeLabel]
			if scheme == "" {
				scheme = "http"
			}

			u := url.URL{
				Scheme: scheme,
				Host:   net.JoinHostPort(service.ServiceAddress, strconv.Itoa(service.ServicePort)),
			}

			targets = append(targets, Target{
				Address: u.String(),
				Path:    tags[pathLabel],
			})
		}
	}

	return targets, nil
}

// Name returns "consul". This method is used to implement the operation.Check interface for use in health checks.
func (cs *ConsulSource) Name() string {
	return "consul"
}

// Check attempts to list services within consul. This method is used to implement the operation.Check interface for use
// in health checks.
func (cs *ConsulSource) Check(ctx context.Context) error {
	options := &api.QueryOptions{
		Filter: cs.filter,
	}

	_, _, err := cs.client.Catalog().Services(options.WithContext(ctx))
	return err
}
