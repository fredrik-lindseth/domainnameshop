// Package domainnameshop registers the Domainname.shop DNS provider with Caddy.
package domainnameshop

import (
	"fmt"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/libdns/domainnameshop"
)

// Provider lets Caddy read and manipulate DNS records hosted by this DNS provider.
type Provider struct{ *domainnameshop.Provider }

func init() {
	caddy.RegisterModule(Provider{})
}

// CaddyModule returns the Caddy module information.
func (Provider) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "dns.providers.domainnameshop",
		New: func() caddy.Module { return &Provider{new(domainnameshop.Provider)} },
	}
}

// Provision sets up the module. Implements caddy.Provisioner.
func (p *Provider) Provision(ctx caddy.Context) error {
	p.initProvider()
	replacer := caddy.NewReplacer()
	for _, field := range []struct {
		name  string
		value *string
	}{{"API token", &p.APIToken}, {"API secret", &p.APISecret}} {
		value, err := replacer.ReplaceOrErr(*field.value, true, true)
		if err != nil {
			// Do not include the input or replacement error: credentials are secrets.
			return fmt.Errorf("%s contains an empty or unknown placeholder", field.name)
		}
		*field.value = value
	}
	return p.Validate()
}

// Validate rejects incomplete credentials for both JSON and Caddyfile configs.
func (p *Provider) Validate() error {
	if p.Provider == nil || strings.TrimSpace(p.APIToken) == "" {
		return fmt.Errorf("missing API token")
	}
	if strings.TrimSpace(p.APISecret) == "" {
		return fmt.Errorf("missing API secret")
	}
	return nil
}

func (p *Provider) initProvider() {
	if p.Provider == nil {
		p.Provider = new(domainnameshop.Provider)
	}
}

// UnmarshalCaddyfile sets up the DNS provider from Caddyfile tokens. Syntax:
//
//	domainnameshop [<api_token>] [<api_secret>] {
//	    api_token <api_token>
//	    api_secret <api_secret>
//	}
func (p *Provider) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	p.initProvider()
	for d.Next() {
		if d.NextArg() {
			p.Provider.APIToken = d.Val()
		}
		if d.NextArg() {
			p.Provider.APISecret = d.Val()
		}
		if d.NextArg() {
			return d.ArgErr()
		}
		for nesting := d.Nesting(); d.NextBlock(nesting); {
			switch d.Val() {
			case "api_token":
				if p.Provider.APIToken != "" {
					return d.Err("API token already set")
				}
				if !d.AllArgs(&p.APIToken) {
					return d.ArgErr()
				}
			case "api_secret":
				if p.Provider.APISecret != "" {
					return d.Err("API secret already set")
				}
				if !d.AllArgs(&p.APISecret) {
					return d.ArgErr()
				}
			default:
				return d.Errf("unrecognized subdirective '%s'", d.Val())
			}
		}
	}
	if err := p.Validate(); err != nil {
		return d.Err(err.Error())
	}
	return nil
}

// Interface guards
var (
	_ caddyfile.Unmarshaler = (*Provider)(nil)
	_ caddy.Provisioner     = (*Provider)(nil)
	_ caddy.Validator       = (*Provider)(nil)
)
