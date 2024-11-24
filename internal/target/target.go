// Package target provides types for managing sources of applications that can be scraped.
package target

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
