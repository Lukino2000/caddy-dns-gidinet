package gidinet

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddytls"

	libgidinet "github.com/Lukino2000/libdns-gidinet"
)

func init() {
	caddy.RegisterModule(Provider{})
}

type Provider struct {
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	CoreEndpoint string `json:"core_endpoint,omitempty"`
	DNSEndpoint  string `json:"dns_endpoint,omitempty"`
}

func (Provider) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "dns.providers.gidinet",
		New: func() caddy.Module { return new(Provider) },
	}
}

func (p *Provider) Provision(ctx caddy.Context) error {
	if p.Username == "" || p.Password == "" {
		return fmt.Errorf("gidinet: username e password sono obbligatori")
	}

	return nil
}

func (p Provider) GetDNSProvider() (caddytls.DNSProvider, error) {
	return &libgidinet.Provider{
		Username:     p.Username,
		Password:     p.Password,
		CoreEndpoint: p.CoreEndpoint,
		DNSEndpoint:  p.DNSEndpoint,
	}, nil
}

func (p *Provider) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "username":
				args := d.RemainingArgs()
				if len(args) != 1 {
					return d.ArgErr()
				}
				p.Username = args[0]
			case "password":
				args := d.RemainingArgs()
				if len(args) != 1 {
					return d.ArgErr()
				}
				p.Password = args[0]
			case "core_endpoint":
				args := d.RemainingArgs()
				if len(args) != 1 {
					return d.ArgErr()
				}
				p.CoreEndpoint = args[0]
			case "dns_endpoint":
				args := d.RemainingArgs()
				if len(args) != 1 {
					return d.ArgErr()
				}
				p.DNSEndpoint = args[0]
			default:
				return d.Errf("parametro non riconosciuto: %s", d.Val())
			}
		}
	}

	return nil
}

var _ caddy.Provisioner = (*Provider)(nil)
var _ caddyfile.Unmarshaler = (*Provider)(nil)
