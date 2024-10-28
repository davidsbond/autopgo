package profile

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/davidsbond/autopgo/internal/closers"
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

// NewClient returns a new instance of the Client type that makes HTTP requests to the provided base URL.
func NewClient(baseURL string) *Client {
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

func bodyToError(body io.Reader) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("could not read body: %w", err)
	}

	return errors.New(string(data))
}
