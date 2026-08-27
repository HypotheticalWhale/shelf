// Package blobstore uploads files to Vercel Blob.
//
// Artwork is re-hosted rather than hot-linked: sources disappear, rate-limit or
// block cross-origin use, and a catalogue that depends on somebody else's
// server is a catalogue that breaks quietly.
package blobstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const endpoint = "https://blob.vercel-storage.com/"

type Client struct {
	token string
	http  *http.Client
}

// New reads the token from BLOB_READ_WRITE_TOKEN.
func New() *Client {
	return &Client{
		token: os.Getenv("BLOB_READ_WRITE_TOKEN"),
		http:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) Configured() bool { return c.token != "" }

type putResponse struct {
	URL         string `json:"url"`
	Pathname    string `json:"pathname"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

// Put stores body at pathname and returns its public URL.
//
// The pathname is used verbatim so a re-run overwrites the same object rather
// than accumulating copies, which keeps the store the same size however often
// the pipeline runs.
func (c *Client) Put(ctx context.Context, pathname, contentType string, body []byte) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("BLOB_READ_WRITE_TOKEN is not set")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint+pathname, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("authorization", "Bearer "+c.token)
	req.Header.Set("x-content-type", contentType)
	req.Header.Set("x-add-random-suffix", "0")
	req.Header.Set("x-content-disposition", "inline")
	// A cover does not change once stored, so let it cache for a year.
	req.Header.Set("x-cache-control-max-age", "31536000")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload %s: %w", pathname, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return "", fmt.Errorf("upload %s returned %d: %s", pathname, resp.StatusCode, msg)
	}

	var out putResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode upload response: %w", err)
	}
	return out.URL, nil
}
