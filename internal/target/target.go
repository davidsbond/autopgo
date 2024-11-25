// Package target provides types for managing sources of applications that can be scraped.
package target

import "strings"

type (
	// The Target type describes individual instances of an application that can be scraped.
	Target struct {
		// The target address, should include scheme, host & port.
		Address string `json:"address"`
		// The path to the pprof profile endpoint, including leading slash. Defaults to /debug/pprof/profile if
		// unset.
		Path string `json:"path"`
	}
)

const (
	scrapeLabel = "autopgo.scrape"
	appLabel    = "autopgo.scrape.app"
	portLabel   = "autopgo.scrape.port"
	pathLabel   = "autopgo.scrape.path"
	schemeLabel = "autopgo.scrape.scheme"
)

func tagsToMap(tags []string) map[string]string {
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
