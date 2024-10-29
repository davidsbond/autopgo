package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/davidsbond/autopgo/internal/closers"
	"github.com/davidsbond/autopgo/internal/logger"
)

type (
	// The Client type is used to interact with the profile server.
	Client struct {
		baseURL string
		http    *http.Client
	}
)

var (
	// ErrNotExist is the error used to indicate a specified profile does not exist.
	ErrNotExist = errors.New("does not exist")
)

// New returns a new instance of the Client type that makes HTTP requests to the provided base URL.
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: time.Minute,
		},
	}
}

// Upload the contents of an application's profile to the profile server.
func (c *Client) Upload(ctx context.Context, app string, r io.Reader) error {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return err
	}

	u.Path = path.Join("/api", "profile", app)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return err
	}

	logger.FromContext(ctx).With(
		slog.String("http.url", req.URL.String()),
		slog.String("http.method", req.Method),
	).DebugContext(ctx, "performing HTTP request")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer closers.Close(ctx, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return bodyToError(resp.Body)
	}

	return nil
}

// Download the profile for a specified application, writing its contents to the given io.Writer implementation. Returns
// ErrNotExist if no profile exists for the application.
func (c *Client) Download(ctx context.Context, app string, w io.Writer) error {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return err
	}

	u.Path = path.Join("/api", "profile", app)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	logger.FromContext(ctx).With(
		slog.String("http.url", req.URL.String()),
		slog.String("http.method", req.Method),
	).DebugContext(ctx, "performing HTTP request")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer closers.Close(ctx, resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		break
	case http.StatusNotFound:
		return ErrNotExist
	default:
		return bodyToError(resp.Body)
	}

	if _, err = io.Copy(w, resp.Body); err != nil {
		return err
	}

	return nil
}

// ProfileAndUpload profiles the provided src URL for the given duration. It then uploads the profile to the server
// using the given application name.
func (c *Client) ProfileAndUpload(ctx context.Context, app, src string, duration time.Duration) error {
	u, err := url.Parse(src)
	if err != nil {
		return err
	}

	u.RawQuery = "seconds=" + strconv.FormatFloat(duration.Seconds(), 'g', -1, 64)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	logger.FromContext(ctx).With(
		slog.String("http.url", req.URL.String()),
		slog.String("http.method", req.Method),
	).DebugContext(ctx, "performing HTTP request")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}

	defer closers.Close(ctx, resp.Body)
	if resp.StatusCode != http.StatusOK {

		return bodyToError(resp.Body)
	}

	return c.Upload(ctx, app, resp.Body)
}

func bodyToError(body io.Reader) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("could not read body: %w", err)
	}

	return errors.New(string(data))
}
