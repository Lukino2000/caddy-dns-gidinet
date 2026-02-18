package gidinet

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

func init() {
	caddy.RegisterModule(Provider{})
}

// CaddyModule returns the Caddy module information.
func (Provider) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "dns.providers.gidinet",
		New: func() caddy.Module {
			return &Provider{}
		},
	}
}

// Provision sets up the provider. Called by Caddy during module provisioning.
func (p *Provider) Provision(ctx caddy.Context) error {
	// Set the log function to use Caddy's logger
	logger := ctx.Logger()
	p.Log = func(msg string, args ...interface{}) {
		logger.Sugar().Infof(msg, args...)
	}

	repl := caddy.NewReplacer()
	g.Username = repl.ReplaceAll(g.Username,"")
	g.Password = repl.ReplaceAll(g.Password,"")

	p.log("Gidinet DNS provider provisioned for user %q", p.Username)
	return nil
}

// UnmarshalCaddyfile parses the Caddyfile tokens for this provider.
//
// Caddyfile syntax:
//
//	gidinet {
//	    username <username>
//	    password <password>
//	    endpoint <endpoint>  # optional
//	}
func (p *Provider) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "username":
				if !d.NextArg() {
					return d.ArgErr()
				}
				p.Username = d.Val()
			case "password":
				if !d.NextArg() {
					return d.ArgErr()
				}
				p.Password = d.Val()
			case "endpoint":
				if !d.NextArg() {
					return d.ArgErr()
				}
				p.Endpoint = d.Val()
			default:
				return d.Errf("unrecognized subdirective: %s", d.Val())
			}
		}
	}
	return nil
}

// Interface guards for Caddy module interfaces.
var (
	_ caddy.Module          = (*Provider)(nil)
	_ caddy.Provisioner     = (*Provider)(nil)
	_ caddyfile.Unmarshaler = (*Provider)(nil)
)
