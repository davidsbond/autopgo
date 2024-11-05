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

	"github.com/davidsbond/autopgo/internal/api"
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
	m.HandleFunc("GET /api/profile", h.List)
	m.HandleFunc("DELETE /api/profile/{app}", h.Delete)
}

type (
	// The UploadResponse type is the response given when a profile has been uploaded.
	UploadResponse struct {
		// The location in blob storage the profile is stored at.
		Key string `json:"key"`
	}
)

// Upload handles an inbound HTTP request containing a pprof profile for a given application. The profile is parsed
// uploaded to blob storage and an event is published.
func (h *HTTPController) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	app := r.PathValue("app")
	if !IsValidAppName(app) {
		api.ErrorResponse(ctx, w, "invalid app name", http.StatusBadRequest)
		return
	}

	p, err := profile.Parse(r.Body)
	if err != nil {
		api.ErrorResponse(ctx, w, err.Error(), http.StatusBadRequest)
		return
	}

	now := time.Now().UnixNano()
	key := path.Join(app, "staging", strconv.FormatInt(now, 10))
	writer, err := h.blobs.NewWriter(ctx, key)
	if err != nil {
		api.ErrorResponse(ctx, w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = p.Write(writer); err != nil {
		api.ErrorResponse(ctx, w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = writer.Close(); err != nil {
		api.ErrorResponse(ctx, w, err.Error(), http.StatusInternalServerError)
		return
	}

	payload := UploadedEvent{
		App:        app,
		ProfileKey: key,
	}

	if err = h.events.Write(ctx, payload); err != nil {
		api.ErrorResponse(ctx, w, err.Error(), http.StatusInternalServerError)
		return
	}

	api.Respond(ctx, w, http.StatusCreated, UploadResponse{Key: key})
}

// Download handles an inbound HTTP request to download a pprof profile for the application specified within the
// URL path.
func (h *HTTPController) Download(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	app := r.PathValue("app")
	if !IsValidAppName(app) {
		api.ErrorResponse(ctx, w, "invalid app name", http.StatusBadRequest)
		return
	}

	key := path.Join(app, "default.pgo")

	reader, err := h.blobs.NewReader(ctx, key)
	switch {
	case errors.Is(err, blob.ErrNotExist):
		api.ErrorResponse(ctx, w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	case err != nil:
		api.ErrorResponse(ctx, w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer closers.Close(ctx, reader)
	if _, err = io.Copy(w, reader); err != nil {
		api.ErrorResponse(ctx, w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type (
	// The ListResponse type is the response given when listing profiles.
	ListResponse struct {
		Profiles []Profile `json:"profiles"`
	}

	// The Profile type describes a single profile stored by the server.
	Profile struct {
		// The location in blob storage of the profile.
		Key string `json:"key"`
		// The profile size in bytes.
		Size int64 `json:"size"`
		// When the profile was last modified.
		LastModified time.Time `json:"lastModified"`
	}
)

// List handles an inbound HTTP request to list all profiles stored by the server.
func (h *HTTPController) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	profiles := make([]Profile, 0)
	for item, err := range h.blobs.List(ctx, IsMergedProfile()) {
		if err != nil {
			api.ErrorResponse(ctx, w, err.Error(), http.StatusInternalServerError)
			return
		}

		profiles = append(profiles, Profile{
			Key:          path.Dir(item.Key),
			Size:         item.Size,
			LastModified: item.LastModified,
		})
	}

	api.Respond(ctx, w, http.StatusOK, ListResponse{Profiles: profiles})
}

type (
	// The DeleteResponse type is the response given when a profile has been deleted.
	DeleteResponse struct{}
)

// Delete handles an inbound HTTP request to delete the profile for an application. It will also delete any profiles
// awaiting merge for the application.
func (h *HTTPController) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	app := r.PathValue("app")

	if !IsValidAppName(app) {
		api.ErrorResponse(ctx, w, "invalid app name", http.StatusBadRequest)
		return
	}

	exists, err := h.blobs.Exists(ctx, path.Join(app, "default.pgo"))
	switch {
	case err != nil:
		api.ErrorResponse(ctx, w, err.Error(), http.StatusInternalServerError)
		return
	case !exists:
		api.ErrorResponse(ctx, w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	for object, err := range h.blobs.List(ctx, IsApplication(app)) {
		if err != nil {
			api.ErrorResponse(ctx, w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = h.blobs.Delete(ctx, object.Key)
		switch {
		case errors.Is(err, blob.ErrNotExist):
			continue
		case err != nil:
			api.ErrorResponse(ctx, w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	api.Respond(ctx, w, http.StatusOK, DeleteResponse{})
}
