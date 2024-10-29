// Package profile contains types responsible for the scraping, uploading & merging of pprof profiles from Go applications.
package profile

import (
	"errors"
	"io"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/google/pprof/profile"

	"github.com/davidsbond/autopgo/internal/blob"
	"github.com/davidsbond/autopgo/internal/closers"
)

type (
	// The HTTPController type is used to handle inbound requests to upload and download application profiles.
	HTTPController struct {
		blobs  BlobRepository
		events EventWriter
	}
)

// NewHTTPController returns a new instance of the HTTPController type that will read and write profiles via the
// given BlobRepository implementation and publish events via the EventWriter implementation.
func NewHTTPController(blobs BlobRepository, events EventWriter) *HTTPController {
	return &HTTPController{
		blobs:  blobs,
		events: events,
	}
}

// Register HTTP endpoints onto the http.ServeMux.
func (h *HTTPController) Register(m *http.ServeMux) {
	m.HandleFunc("POST /api/profile/{app}", h.Upload)
	m.HandleFunc("GET /api/profile/{app}", h.Download)
}

// Upload handles an inbound HTTP request containing a pprof profile for a given application. The profile is parsed
// uploaded to blob storage and an event is published.
func (h *HTTPController) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	app := r.PathValue("app")
	if !IsValidAppName(app) {
		http.Error(w, "invalid app name", http.StatusBadRequest)
		return
	}

	p, err := profile.Parse(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	now := time.Now().Unix()
	key := path.Join(app, "staging", strconv.FormatInt(now, 10))
	writer, err := h.blobs.NewWriter(ctx, key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = p.Write(writer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = writer.Close(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	payload := UploadedEvent{
		App:        app,
		ProfileKey: key,
	}

	if err = h.events.Write(ctx, payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Download handles an inbound HTTP request to download a pprof profile for the application specified within the
// URL path.
func (h *HTTPController) Download(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	app := r.PathValue("app")
	if !IsValidAppName(app) {
		http.Error(w, "invalid app name", http.StatusBadRequest)
		return
	}

	key := path.Join(app, "default.pgo")

	reader, err := h.blobs.NewReader(ctx, key)
	switch {
	case errors.Is(err, blob.ErrNotExist):
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer closers.Close(ctx, reader)
	if _, err = io.Copy(w, reader); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
