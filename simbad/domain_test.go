package simbad

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests are offline: they exercise the URI driver's pure string functions
// and the host wiring (domain info, register). The client's HTTP behaviour is
// covered in simbad_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "simbad" {
		t.Errorf("Scheme = %q, want simbad", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "simbad" {
		t.Errorf("Identity.Binary = %q, want simbad", info.Identity.Binary)
	}
}

func TestDomainRegister(t *testing.T) {
	// kit.Open uses the registered Domain from the init() call.
	_, err := kit.Open()
	if err != nil {
		t.Fatalf("kit.Open: %v", err)
	}
}

func TestQueryObjects_handler(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"metadata": []map[string]interface{}{
				{"name": "main_id", "datatype": "CHAR"},
				{"name": "ra", "datatype": "DOUBLE"},
				{"name": "dec", "datatype": "DOUBLE"},
				{"name": "otype_txt", "datatype": "CHAR"},
			},
			"data": [][]interface{}{
				{"M 31", 10.684, 41.269, "G"},
				{"M 33", 23.462, 30.660, "G"},
			},
		})
	}))
	defer ts.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	var got []*Object
	err := queryObjects(context.Background(), queryInput{
		Type:   "G",
		Top:    5,
		Client: c,
	}, func(o *Object) error {
		got = append(got, o)
		return nil
	})
	if err != nil {
		t.Fatalf("queryObjects: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d objects, want 2", len(got))
	}
	if got[0].MainID != "M 31" {
		t.Errorf("got[0].MainID = %q, want M 31", got[0].MainID)
	}
}

func TestGetObject_handler(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"metadata": []map[string]interface{}{
				{"name": "main_id", "datatype": "CHAR"},
				{"name": "ra", "datatype": "DOUBLE"},
				{"name": "dec", "datatype": "DOUBLE"},
				{"name": "otype_txt", "datatype": "CHAR"},
				{"name": "z_value", "datatype": "DOUBLE"},
				{"name": "rvz_radvel", "datatype": "DOUBLE"},
			},
			"data": [][]interface{}{
				{"M 31", 10.684, 41.269, "G", -0.001001, -300.0},
			},
		})
	}))
	defer ts.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	var got []*Object
	err := getObject(context.Background(), objectInput{
		Name:   "Andromeda",
		Client: c,
	}, func(o *Object) error {
		got = append(got, o)
		return nil
	})
	if err != nil {
		t.Fatalf("getObject: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d objects, want 1", len(got))
	}
	if got[0].MainID != "M 31" {
		t.Errorf("got[0].MainID = %q, want M 31", got[0].MainID)
	}
	if got[0].RadVel != -300.0 {
		t.Errorf("got[0].RadVel = %v, want -300.0", got[0].RadVel)
	}
}

func TestListStars_handler(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"metadata": []map[string]interface{}{
				{"name": "main_id", "datatype": "CHAR"},
				{"name": "ra", "datatype": "DOUBLE"},
				{"name": "dec", "datatype": "DOUBLE"},
				{"name": "otype_txt", "datatype": "CHAR"},
			},
			"data": [][]interface{}{
				{"* alf Cen A", 219.9009, -60.8353, "*"},
				{"* alf Cen B", 219.9009, -60.8353, "*"},
			},
		})
	}))
	defer ts.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	var got []*Object
	err := listStars(context.Background(), starsInput{
		Top:    2,
		Client: c,
	}, func(o *Object) error {
		got = append(got, o)
		return nil
	})
	if err != nil {
		t.Fatalf("listStars: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d objects, want 2", len(got))
	}
	if got[0].ObjectType != "*" {
		t.Errorf("got[0].ObjectType = %q, want *", got[0].ObjectType)
	}
}

func TestRawQuery_handler(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"metadata": []map[string]interface{}{
				{"name": "main_id", "datatype": "CHAR"},
				{"name": "ra", "datatype": "DOUBLE"},
				{"name": "dec", "datatype": "DOUBLE"},
				{"name": "otype_txt", "datatype": "CHAR"},
			},
			"data": [][]interface{}{
				{"SN 1987A", 83.866, -69.269, "SNR"},
			},
		})
	}))
	defer ts.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	c := NewClient(cfg)

	var got []*Object
	err := rawQuery(context.Background(), tapInput{
		Query:  "SELECT main_id,ra,dec,otype_txt FROM basic WHERE main_id='SN 1987A'",
		Client: c,
	}, func(o *Object) error {
		got = append(got, o)
		return nil
	})
	if err != nil {
		t.Fatalf("rawQuery: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d objects, want 1", len(got))
	}
	if got[0].MainID != "SN 1987A" {
		t.Errorf("got[0].MainID = %q, want SN 1987A", got[0].MainID)
	}
}

func TestEscapeSQL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"M 31", "M 31"},
		{"O'Brien", "O''Brien"},
		{"it's", "it''s"},
	}
	for _, tc := range cases {
		got := escapeSQL(tc.in)
		if got != tc.want {
			t.Errorf("escapeSQL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
