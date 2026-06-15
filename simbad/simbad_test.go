package simbad

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// tapResponse is a helper to build a mock TAP JSON response.
func tapResponse(data [][]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"metadata": []map[string]interface{}{
			{"name": "main_id", "datatype": "CHAR"},
			{"name": "ra", "datatype": "DOUBLE"},
			{"name": "dec", "datatype": "DOUBLE"},
			{"name": "otype_txt", "datatype": "CHAR"},
			{"name": "z_value", "datatype": "DOUBLE"},
			{"name": "rvz_radvel", "datatype": "DOUBLE"},
		},
		"data": data,
	}
}

func newTestClient(ts *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0 // no pacing in tests
	return NewClient(cfg)
}

func TestQueryTAP_basic(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		q := r.URL.Query().Get("QUERY")
		if q == "" {
			t.Error("QUERY parameter missing")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tapResponse([][]interface{}{
			{"M 31", 10.684, 41.269, "G", 0.0, 0.0},
			{"M 33", 23.462, 30.660, "G", 0.0, 0.0},
		}))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	objs, err := c.QueryTAP(context.Background(), "SELECT TOP 2 main_id,ra,dec,otype_txt FROM basic")
	if err != nil {
		t.Fatalf("QueryTAP: %v", err)
	}
	if len(objs) != 2 {
		t.Fatalf("got %d objects, want 2", len(objs))
	}
	if objs[0].MainID != "M 31" {
		t.Errorf("objs[0].MainID = %q, want M 31", objs[0].MainID)
	}
	if objs[0].RA != 10.684 {
		t.Errorf("objs[0].RA = %v, want 10.684", objs[0].RA)
	}
	if objs[0].Dec != 41.269 {
		t.Errorf("objs[0].Dec = %v, want 41.269", objs[0].Dec)
	}
	if objs[0].ObjectType != "G" {
		t.Errorf("objs[0].ObjectType = %q, want G", objs[0].ObjectType)
	}
}

func TestQueryTAP_withRedshift(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tapResponse([][]interface{}{
			{"3C 273", 187.278, 2.052, "QSO", 0.158, 47800.0},
		}))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	objs, err := c.QueryTAP(context.Background(), "SELECT main_id,ra,dec,otype_txt,z_value,rvz_radvel FROM basic WHERE main_id='3C 273'")
	if err != nil {
		t.Fatalf("QueryTAP: %v", err)
	}
	if len(objs) != 1 {
		t.Fatalf("got %d objects, want 1", len(objs))
	}
	if objs[0].Redshift != 0.158 {
		t.Errorf("Redshift = %v, want 0.158", objs[0].Redshift)
	}
	if objs[0].RadVel != 47800.0 {
		t.Errorf("RadVel = %v, want 47800.0", objs[0].RadVel)
	}
}

func TestQueryTAP_retriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tapResponse([][]interface{}{
			{"M 31", 10.684, 41.269, "G", 0.0, 0.0},
		}))
	}))
	defer ts.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := NewClient(cfg)

	objs, err := c.QueryTAP(context.Background(), "SELECT TOP 1 main_id,ra,dec,otype_txt FROM basic")
	if err != nil {
		t.Fatalf("QueryTAP: %v", err)
	}
	if len(objs) != 1 {
		t.Fatalf("got %d objects after retries, want 1", len(objs))
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestQueryTAP_emptyResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tapResponse(nil))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	objs, err := c.QueryTAP(context.Background(), "SELECT TOP 0 main_id,ra,dec,otype_txt FROM basic")
	if err != nil {
		t.Fatalf("QueryTAP: %v", err)
	}
	if len(objs) != 0 {
		t.Errorf("got %d objects, want 0", len(objs))
	}
}

func TestEffectiveTop(t *testing.T) {
	cases := []struct{ top, limit, want int }{
		{0, 0, 20},
		{10, 0, 10},
		{0, 5, 5},
		{10, 5, 5},
		{5, 10, 5},
	}
	for _, tc := range cases {
		got := effectiveTop(tc.top, tc.limit)
		if got != tc.want {
			t.Errorf("effectiveTop(%d, %d) = %d, want %d", tc.top, tc.limit, got, tc.want)
		}
	}
}
