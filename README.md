# Gidinet DNS provider for Caddy 2

This package implements the [libdns](https://github.com/libdns/libdns) interfaces for the [Gidinet](https://www.gidinet.com/) DNS API (SOAP), allowing you to manage DNS records via Caddy's automatic HTTPS (ACME DNS-01 challenge).

---
## Caddy module name

dns.providers.gidinet


---
## Configuration

### Caddyfile

```caddyfile
tls {
    dns gidinet {
        username {env.GIDINET_USERNAME}
        password {env.GIDINET_PASSWORD}
    }
}
```

### JSON

```json
{
    "module": "dns.providers.gidinet",
    "username": "your_username",
    "password": "your_password"
}
```

---
## Building Caddy with this module

```bash
xcaddy build --with github.com/Lukino2000/caddy-dns-gidinet
```

---
## Testing

```bash
cd example
cp .env.example .env
# Edit .env with your real credentials
go run .
```
