// Package simbad is the library behind the simbad command line:
// the HTTP client, request shaping, and the typed data models for the
// SIMBAD Astronomical Database at simbad.u-strasbg.fr.
//
// The Client here is the spine every command shares. It sets a real
// User-Agent, paces requests so a busy session stays polite, and retries the
// transient failures (429 and 5xx) that any public site throws under load.
package simbad

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Host is the site this client talks to, and the host the URI driver in
// domain.go claims.
const Host = "simbad.u-strasbg.fr"

// tapURL is the TAP/ADQL synchronous query endpoint.
const tapURL = "https://simbad.u-strasbg.fr/simbad/sim-tap/sync"

// DefaultUserAgent identifies the client to SIMBAD. A real, honest
// User-Agent is both polite and the thing most likely to keep you unblocked.
const DefaultUserAgent = "simbad-cli/0.1 (tamnd87@gmail.com)"

// Object is one record from the SIMBAD basic table.
type Object struct {
	MainID     string  `json:"main_id"`
	RA         float64 `json:"ra"`
	Dec        float64 `json:"dec"`
	ObjectType string  `json:"object_type"`
	Redshift   float64 `json:"redshift,omitempty"`
	RadVel     float64 `json:"radial_velocity,omitempty"`
}

// Config holds all tunables for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns safe, polite defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   tapURL,
		Rate:      1 * time.Second,
		Timeout:   30 * time.Second,
		Retries:   2,
		UserAgent: DefaultUserAgent,
	}
}

// Client talks to SIMBAD over HTTP.
type Client struct {
	cfg  Config
	http *http.Client
	last time.Time
}

// NewClient returns a Client with DefaultConfig settings.
func NewClient(cfg ...Config) *Client {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	return &Client{
		cfg:  c,
		http: &http.Client{Timeout: c.Timeout},
	}
}

// --- TAP query helpers ---

// tapResult holds raw TAP/ADQL response.
type tapResult struct {
	Metadata []struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Data [][]interface{} `json:"data"`
}

// toObjects converts TAP rows to Object slices.
func (r tapResult) toObjects() []Object {
	idx := map[string]int{}
	for i, m := range r.Metadata {
		idx[m.Name] = i
	}
	var objs []Object
	for _, row := range r.Data {
		o := Object{}
		if i, ok := idx["main_id"]; ok && i < len(row) {
			if s, ok := row[i].(string); ok {
				o.MainID = s
			}
		}
		if i, ok := idx["ra"]; ok && i < len(row) {
			if f, ok := row[i].(float64); ok {
				o.RA = f
			}
		}
		if i, ok := idx["dec"]; ok && i < len(row) {
			if f, ok := row[i].(float64); ok {
				o.Dec = f
			}
		}
		if i, ok := idx["otype_txt"]; ok && i < len(row) {
			if s, ok := row[i].(string); ok {
				o.ObjectType = s
			}
		}
		if i, ok := idx["z_value"]; ok && i < len(row) {
			if f, ok := row[i].(float64); ok {
				o.Redshift = f
			}
		}
		if i, ok := idx["rvz_radvel"]; ok && i < len(row) {
			if f, ok := row[i].(float64); ok {
				o.RadVel = f
			}
		}
		objs = append(objs, o)
	}
	return objs
}

// QueryTAP executes an ADQL query and returns the matching objects.
func (c *Client) QueryTAP(ctx context.Context, adql string) ([]Object, error) {
	params := url.Values{}
	params.Set("REQUEST", "doQuery")
	params.Set("LANG", "ADQL")
	params.Set("FORMAT", "json")
	params.Set("QUERY", adql)

	fullURL := c.cfg.BaseURL + "?" + params.Encode()

	body, err := c.get(ctx, fullURL)
	if err != nil {
		return nil, err
	}

	var tr tapResult
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("parse TAP response: %w", err)
	}
	return tr.toObjects(), nil
}

// get fetches url and returns the response body. It paces and retries according
// to the client's settings.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// pace blocks until at least Rate has passed since the previous request.
func (c *Client) pace() {
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// effectiveTop returns the effective TOP N: honours both --top and --limit
// flags, picking the smaller non-zero value; defaults to 20.
func effectiveTop(top, limit int) int {
	switch {
	case top > 0 && limit > 0:
		if top < limit {
			return top
		}
		return limit
	case top > 0:
		return top
	case limit > 0:
		return limit
	default:
		return 20
	}
}
