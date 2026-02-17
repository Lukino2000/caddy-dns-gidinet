package gidinet

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddytls"
	"github.com/libdns/libdns"

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
	return nil
}

func (p Provider) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "username":
				p.Username = d.RemainingArgs()[0]
			case "password":
				p.Password = d.RemainingArgs()[0]
			case "core_endpoint":
				p.CoreEndpoint = d.RemainingArgs()[0]
			case "dns_endpoint":
				p.DNSEndpoint = d.RemainingArgs()[0]
			}
		}
	}

	return nil
}

func (p Provider) GetDNSProvider() (caddytls.DNSProvider, error) {
	return caddytls.DNSProvider(libgidinet.Provider{
		Username:     p.Username,
		Password:     p.Password,
		CoreEndpoint: p.CoreEndpoint,
		DNSEndpoint:  p.DNSEndpoint,
	}), nil
}

var _ caddytls.DNSProvider = (*libgidinet.Provider)(nil)
var _ caddy.Provisioner = (*Provider)(nil)
var _ caddyfile.Unmarshaler = (*Provider)(nil)
