package target

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/nomad/api"
)

type (
	NomadSource struct {
		client *api.Client
		filter string
	}
)

func NewNomadSource(client *api.Client, app string) *NomadSource {
	return &NomadSource{
		client: client,
		filter: fmt.Sprintf(`Tags contains "autopgo.scrape=true" and Tags contains "autopgo.scrape.app=%s"`, app),
	}
}

func (ns *NomadSource) List(ctx context.Context) ([]Target, error) {
	listOpts := &api.QueryOptions{
		Namespace: api.AllNamespacesNamespace,
		Filter:    ns.filter,
	}

	resp, _, err := ns.client.Services().List(listOpts.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	targets := make([]Target, 0)
	for _, entry := range resp {
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
