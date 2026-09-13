package domainnameshop

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/libdns/libdns"
)

func TestUnmarshalCaddyfile(t *testing.T) {
	for _, tc := range []struct {
		name, input, wantErr string
	}{
		{"inline", "domainnameshop token secret", ""},
		{"block", "domainnameshop {\n api_token token\n api_secret secret\n}", ""},
		{"mixed", "domainnameshop token {\n api_secret secret\n}", ""},
		{"missing token", "domainnameshop", "missing API token"},
		{"missing secret", "domainnameshop token", "missing API secret"},
		{"extra inline", "domainnameshop token secret extra", "argument"},
		{"empty token argument", "domainnameshop {\n api_token\n api_secret secret\n}", "argument"},
		{"empty secret argument", "domainnameshop token {\n api_secret\n}", "argument"},
		{"extra token argument", "domainnameshop {\n api_token token extra\n api_secret secret\n}", "argument"},
		{"duplicate token", "domainnameshop token secret {\n api_token other\n}", "already set"},
		{"duplicate secret", "domainnameshop token secret {\n api_secret other\n}", "already set"},
		{"duplicate block", "domainnameshop {\n api_token token\n api_token other\n api_secret secret\n}", "already set"},
		{"unknown", "domainnameshop token secret {\n typo value\n}", "unrecognized"},
		{"nested block", "domainnameshop {\n api_token token {\n ignored value\n }\n api_secret secret\n}", "unrecognized"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p Provider // The public zero value must be safe to configure.
			err := p.UnmarshalCaddyfile(caddyfile.NewTestDispenser(tc.input))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("got %v; want error containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if p.APIToken != "token" || p.APISecret != "secret" {
				t.Fatal("credentials parsed incorrectly")
			}
		})
	}
}

func TestProvision(t *testing.T) {
	t.Setenv("DOMAINNAMESHOP_TEST_TOKEN", "resolved-token")
	t.Setenv("DOMAINNAMESHOP_TEST_SECRET", "resolved-secret")
	t.Setenv("DOMAINNAMESHOP_TEST_EMPTY", "")
	for _, tc := range []struct {
		name, token, secret, wantErr string
	}{
		{"literal", "token", "secret", ""},
		{"environment", "{env.DOMAINNAMESHOP_TEST_TOKEN}", "{env.DOMAINNAMESHOP_TEST_SECRET}", ""},
		{"missing token", "", "secret", "missing API token"},
		{"missing secret", "token", "", "missing API secret"},
		{"whitespace token", "  ", "secret", "missing API token"},
		{"whitespace secret", "token", "\t", "missing API secret"},
		{"empty environment", "prefix-{env.DOMAINNAMESHOP_TEST_EMPTY}", "secret", "API token contains"},
		{"unknown placeholder", "token", "secret-{not.a.placeholder}", "API secret contains"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := Provider{}.CaddyModule().New().(*Provider)
			p.APIToken, p.APISecret = tc.token, tc.secret
			err := p.Provision(caddy.Context{})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("got %v; want %q", err, tc.wantErr)
				}
				if strings.Contains(err.Error(), "secret-") || strings.Contains(err.Error(), "prefix-") {
					t.Fatal("error exposed a credential value")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.name == "environment" && (p.APIToken != "resolved-token" || p.APISecret != "resolved-secret") {
				t.Fatal("environment placeholders were not resolved")
			}
		})
	}
}

func TestZeroValueProvision(t *testing.T) {
	var p Provider
	if err := p.Provision(caddy.Context{}); err == nil {
		t.Fatal("missing credentials must be rejected")
	}
}

func TestModuleJSON(t *testing.T) {
	info, err := caddy.GetModule("dns.providers.domainnameshop")
	if err != nil {
		t.Fatal(err)
	}
	p := info.New().(*Provider)
	if err := json.Unmarshal([]byte(`{"api_token":"token","api_secret":"secret"}`), p); err != nil {
		t.Fatal(err)
	}
	if err := p.Provision(caddy.Context{}); err != nil {
		t.Fatal(err)
	}
	if p.APIToken != "token" || p.APISecret != "secret" {
		t.Fatal("embedded JSON credentials were not decoded")
	}
}

// Keep the DNS-01 interfaces compatible with the selected libdns dependency.
var (
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
