package target

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/nomad/api"

	"github.com/davidsbond/autopgo/internal/logger"
)

type (
	// The NomadSource type is used to source scrapable targets from a HashiCorp Nomad cluster using its services
	// API.
	NomadSource struct {
		client *api.Client
		filter string
	}
)

// NewNomadSource returns a new instance of the NomadSource type that will source targets using the provided Nomad
// client. It will search for services tagged with the provided app name.
func NewNomadSource(client *api.Client, app string) *NomadSource {
	return &NomadSource{
		client: client,
		filter: fmt.Sprintf(`Tags contains "autopgo.scrape=true" and Tags contains "autopgo.scrape.app=%s"`, app),
	}
}

// List all targets within the Nomad cluster matching the application. This method will use the Nomad services API to
// find services that have two main tags: autopgo.scrape=true and autopgo.scrape.app=app. The latter tag should use
// the configured application name as the tag value. A custom path & scheme can be set using the autopgo.scrape.path
// and autopgo.scrape.scheme tags.
func (ns *NomadSource) List(ctx context.Context) ([]Target, error) {
	log := logger.FromContext(ctx)

	listOpts := &api.QueryOptions{
		Namespace: api.AllNamespacesNamespace,
		Filter:    ns.filter,
	}

	log.DebugContext(ctx, "listing nomad services")
	resp, _, err := ns.client.Services().List(listOpts.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	log.
		With(slog.Int("count", len(resp))).
		DebugContext(ctx, "found namespaces with tagged services")

	targets := make([]Target, 0)
	for _, entry := range resp {
		log.With(
			slog.Int("count", len(entry.Services)),
			slog.String("namespace", entry.Namespace),
		).DebugContext(ctx, "found tagged services")

		for _, serviceEntry := range entry.Services {
			getOpts := &api.QueryOptions{
				Namespace: entry.Namespace,
				Filter:    ns.filter,
			}

			services, _, err := ns.client.Services().Get(serviceEntry.ServiceName, getOpts.WithContext(ctx))
			if err != nil {
				return nil, err
			}

			for _, service := range services {
				tags := nomadTagsToMap(service.Tags)

				scheme := tags[schemeLabel]
				if scheme == "" {
					scheme = "http"
				}

				u := url.URL{
					Scheme: scheme,
					Host:   net.JoinHostPort(service.Address, strconv.Itoa(service.Port)),
				}

				targets = append(targets, Target{
					Address: u.String(),
					Path:    tags[pathLabel],
				})
			}
		}
	}

	return targets, nil
}

func nomadTagsToMap(tags []string) map[string]string {
	out := make(map[string]string)
	for _, tag := range tags {
		if !strings.HasPrefix(tag, "autopgo") {
			continue
		}

		parts := strings.SplitN(tag, "=", 2)
		if len(parts) != 2 {
			continue
		}

		out[parts[0]] = parts[1]
	}

	return out
}
