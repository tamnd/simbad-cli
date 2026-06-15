package simbad

import (
	"context"
	"fmt"

	"github.com/tamnd/any-cli/kit"
)

// domain.go exposes SIMBAD as a kit Domain: a driver that a multi-domain
// host (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/simbad-cli/simbad"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host then routes
// simbad:// URIs to the operations Register installs. The same Domain also
// builds the standalone simbad binary (see cli.NewApp), so the binary and a
// host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the SIMBAD driver. It carries no state; the per-run client is
// built by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "simbad",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "simbad",
			Short:  "Query the SIMBAD Astronomical Database.",
			Long: `Query the SIMBAD Astronomical Database at simbad.u-strasbg.fr.

simbad reads millions of astronomical objects (stars, galaxies, nebulae, etc.)
via ADQL/TAP queries over plain HTTPS. No API key required. Output pipes into
jq, grep, or any other tool.`,
			Site: Host,
			Repo: "https://github.com/tamnd/simbad-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "query", Group: "read", List: true,
		Summary: "Query astronomical objects by type (--type G=Galaxy *=Star SNR=Supernova)"}, queryObjects)

	kit.Handle(app, kit.OpMeta{Name: "object", Group: "read", Single: true,
		Summary: "Look up an astronomical object by name or identifier",
		Args:    []kit.Arg{{Name: "name", Help: "object name or identifier (e.g. M31, Andromeda)"}}}, getObject)

	kit.Handle(app, kit.OpMeta{Name: "stars", Group: "read", List: true,
		Summary: "List stars from the SIMBAD catalog"}, listStars)

	kit.Handle(app, kit.OpMeta{Name: "tap", Group: "read", List: true,
		Summary: "Execute a raw ADQL query against SIMBAD",
		Args:    []kit.Arg{{Name: "query", Help: "ADQL SELECT statement"}}}, rawQuery)
}

// newClient builds the client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- inputs ---

type queryInput struct {
	Type   string  `kit:"flag" help:"object type filter (G=Galaxy *=Star SNR=Supernova Remnant)"`
	Top    int     `kit:"flag" help:"max results (TOP N in ADQL)"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type objectInput struct {
	Name   string  `kit:"arg" help:"object name or identifier"`
	Client *Client `kit:"inject"`
}

type starsInput struct {
	Top    int     `kit:"flag" help:"max results"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type tapInput struct {
	Query  string  `kit:"arg" help:"ADQL SELECT query"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func queryObjects(ctx context.Context, in queryInput, emit func(*Object) error) error {
	n := effectiveTop(in.Top, in.Limit)
	adql := fmt.Sprintf("SELECT TOP %d main_id,ra,dec,otype_txt FROM basic", n)
	if in.Type != "" {
		adql += fmt.Sprintf(" WHERE otype_txt LIKE '%%%s%%'", escapeLike(in.Type))
	}
	adql += " ORDER BY main_id"

	objs, err := in.Client.QueryTAP(ctx, adql)
	if err != nil {
		return err
	}
	for i := range objs {
		if err := emit(&objs[i]); err != nil {
			return err
		}
	}
	return nil
}

func getObject(ctx context.Context, in objectInput, emit func(*Object) error) error {
	adql := fmt.Sprintf(
		"SELECT main_id,ra,dec,otype_txt,z_value,rvz_radvel FROM basic JOIN ident ON basic.oid=ident.oidref WHERE ident.id='%s'",
		escapeSQL(in.Name),
	)
	objs, err := in.Client.QueryTAP(ctx, adql)
	if err != nil {
		return err
	}
	for i := range objs {
		if err := emit(&objs[i]); err != nil {
			return err
		}
	}
	return nil
}

func listStars(ctx context.Context, in starsInput, emit func(*Object) error) error {
	n := effectiveTop(in.Top, in.Limit)
	adql := fmt.Sprintf("SELECT TOP %d main_id,ra,dec,otype_txt FROM basic WHERE otype_txt='*' ORDER BY main_id", n)

	objs, err := in.Client.QueryTAP(ctx, adql)
	if err != nil {
		return err
	}
	for i := range objs {
		if err := emit(&objs[i]); err != nil {
			return err
		}
	}
	return nil
}

func rawQuery(ctx context.Context, in tapInput, emit func(*Object) error) error {
	objs, err := in.Client.QueryTAP(ctx, in.Query)
	if err != nil {
		return err
	}
	for i := range objs {
		if err := emit(&objs[i]); err != nil {
			return err
		}
	}
	return nil
}

// --- helpers ---

// escapeSQL escapes single quotes in a string for use in an ADQL literal.
// SIMBAD's ADQL follows standard SQL escaping: a literal ' becomes ''.
func escapeSQL(s string) string {
	out := make([]byte, 0, len(s)+4)
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			out = append(out, '\'', '\'')
		} else {
			out = append(out, s[i])
		}
	}
	return string(out)
}

// escapeLike escapes ADQL LIKE wildcards inside a user-supplied type string.
func escapeLike(s string) string {
	return escapeSQL(s)
}
