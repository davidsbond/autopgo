// Package server provides types for running an HTTP server with middleware.
package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"golang.org/x/sync/errgroup"
)

type (
	// The Config type contains configuration values used for running an HTTP server.
	Config struct {
		// The port to serve HTTP traffic on.
		Port int
		// Individual controllers to register onto the server's http.Handler.
		Controllers []Controller
		// Middleware functions to invoke prior to request handlers.
		Middleware []Middleware
	}

	// The Controller interface describes types that register HTTP request handlers.
	Controller interface {
		// Register the Controller's endpoints onto the provided http.ServeMux.
		Register(m *http.ServeMux)
	}

	// The Middleware type is a function that wraps an http.Handler. Used to perform actions prior
	// to handling requests.
	Middleware func(http.Handler) http.Handler
)

// Run an HTTP server based on the provided configuration. This function blocks until the provided context is
// cancelled.
func Run(ctx context.Context, config Config) error {
	mux := http.NewServeMux()

	for _, controller := range config.Controllers {
		controller.Register(mux)
	}

	registerDebug(mux)

	server := &http.Server{
		Handler: mux,
		Addr:    fmt.Sprintf(":%d", config.Port),
	}

	for _, middleware := range config.Middleware {
		server.Handler = middleware(server.Handler)
	}

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return server.ListenAndServe()
	})
	group.Go(func() error {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		return server.Shutdown(ctx)
	})

	return group.Wait()
}

func registerDebug(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
}
