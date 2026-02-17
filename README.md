# caddy-dns-gidinet

Provider DNS per Caddy (ACME DNS-01) basato su GiDiNet.

Questo modulo usa il provider libdns:  
**github.com/Lukino2000/libdns-gidinet**

---

## Installazione

Compila Caddy con `xcaddy` includendo il modulo:

```bash
go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest

xcaddy build --with github.com/Lukino2000/caddy-dns-gidinet
```

# Configurazione Caddyfile

## Esempio base

```caddyfile
example.com {
	tls {
		dns gidinet {
			username gdn5377
			password tua_password
		}
	}
	respond "ok"
}
```

## Esempio con variabili ambiente

```caddyfile
example.com {
	tls {
		dns gidinet {
			username {$GIDINET_USERNAME}
			password {$GIDINET_PASSWORD}
		}
	}
	respond "ok"
}
```

## Note
La password viene inviata come Base64 (gestita dal provider).
Hostname relativo: per record TXT ACME viene usato _acme-challenge.

## Endpoint di default:
Core: https://api.quickservicebox.com/API/Beta/CoreAPI.asmx
DNS:  https://api.quickservicebox.com/API/Beta/DNSAPI.asmx

Se servono endpoint diversi, verranno supportati nei prossimi rilasci.

## Licenza
MIT