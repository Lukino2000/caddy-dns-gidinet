package gidinet

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/libdns/libdns"
)

// allowedTTLs contains the TTL values (in seconds) permitted by the Gidinet API,
// sorted in ascending order.
var allowedTTLs = []int{
	60, 300, 600, 900, 1800, 2700, 3600, 7200,
	14400, 28800, 43200, 64800, 86400, 172800,
}

// LogFunc is the signature for the logging callback.
// If set on the Provider, it will be called for every significant operation.
type LogFunc func(msg string, args ...interface{})

// Provider implements the libdns interfaces for the Gidinet DNS API.
type Provider struct {
	// Username is the Gidinet account username.
	Username string `json:"username"`

	// Password is the Gidinet account password (plain text; will be base64-encoded before sending).
	Password string `json:"password"`

	// Endpoint optionally overrides the default API endpoint.
	// Leave empty to use the default: https://api.quickservicebox.com/API/Beta/DNSAPI.asmx
	Endpoint string `json:"endpoint,omitempty"`

	// Log is an optional logging function. If nil, no logging is performed.
	Log LogFunc `json:"-"`

	client     *soapClient
	clientOnce sync.Once
	mu         sync.Mutex
}

// getClient lazily initializes and returns the SOAP client.
func (p *Provider) getClient() *soapClient {
	p.clientOnce.Do(func() {
		passwordB64 := base64.StdEncoding.EncodeToString([]byte(p.Password))
		p.client = newSOAPClient(p.Username, passwordB64, p.Endpoint)
	})
	return p.client
}

// log calls the Log function if it is set.
func (p *Provider) log(msg string, args ...interface{}) {
	if p.Log != nil {
		p.Log(msg, args...)
	}
}

// normalizeTTL rounds the given duration down to the nearest allowed TTL value.
// If the duration is smaller than the smallest allowed TTL, an error is returned.
// A zero or negative duration defaults to the smallest allowed TTL.
func normalizeTTL(d time.Duration) (uint, error) {
	seconds := int(d.Seconds())
	if seconds <= 0 {
		// Default to the smallest allowed TTL when no TTL is specified
		return uint(allowedTTLs[0]), nil
	}

	// Find the largest allowed TTL that is <= seconds (round down)
	result := -1
	for _, ttl := range allowedTTLs {
		if ttl <= seconds {
			result = ttl
		} else {
			break
		}
	}

	if result == -1 {
		return 0, fmt.Errorf("TTL %d seconds is smaller than the minimum allowed value (%d seconds)", seconds, allowedTTLs[0])
	}

	return uint(result), nil
}

// zoneToDomain converts a libdns zone (e.g. "example.com.") to a bare domain name (e.g. "example.com").
func zoneToDomain(zone string) string {
	return strings.TrimSuffix(strings.TrimSpace(zone), ".")
}

// relativeToFQDN converts a libdns relative record name to the FQDN HostName expected by Gidinet.
// For example, name="_acme-challenge" and domain="example.com" becomes "_acme-challenge.example.com".
// An empty name or "@" means the domain apex, which Gidinet expects as "@".
func relativeToFQDN(name, domain string) string {
	name = strings.TrimSuffix(strings.TrimSpace(name), ".")
	if name == "" || name == "@" {
		return "@"
	}
	// If the name already ends with the domain, return it as-is
	if strings.HasSuffix(name, domain) {
		return name
	}
	return name + "." + domain
}

// fqdnToRelative converts a Gidinet FQDN HostName to a libdns relative record name.
// For example, hostname="_acme-challenge.example.com" and domain="example.com" becomes "_acme-challenge".
// If the hostname equals the domain, it returns "@" (apex).
func fqdnToRelative(hostname, domain string) string {
	hostname = strings.TrimSuffix(strings.TrimSpace(hostname), ".")
	domain = strings.TrimSuffix(strings.TrimSpace(domain), ".")

	if strings.EqualFold(hostname, domain) || hostname == "@" {
		return "@"
	}

	suffix := "." + domain
	if strings.HasSuffix(hostname, suffix) {
		return strings.TrimSuffix(hostname, suffix)
	}

	return hostname
}

// toLibdnsRecord converts a Gidinet DNS record list item to a libdns.Record.
// It returns the appropriate concrete type (Address, TXT, CNAME, MX, NS, SRV, CAA)
// based on the record type. For unknown types, it returns an RR.
func toLibdnsRecord(item dnsRecordListItem, domain string) libdns.Record {
	name := fqdnToRelative(item.HostName, domain)
	ttl := time.Duration(item.TTL) * time.Second

	switch strings.ToUpper(item.RecordType) {
	case "A", "AAAA":
		addr, err := netip.ParseAddr(item.Data)
		if err != nil {
			// Fall back to RR if the IP cannot be parsed
			return libdns.RR{
				Name: name,
				TTL:  ttl,
				Type: item.RecordType,
				Data: item.Data,
			}
		}
		return libdns.Address{
			Name: name,
			TTL:  ttl,
			IP:   addr,
		}

	case "TXT":
		// Strip surrounding quotes if present (Gidinet may return quoted TXT values)
		text := item.Data
		if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
			text = text[1 : len(text)-1]
		}
		return libdns.TXT{
			Name: name,
			TTL:  ttl,
			Text: text,
		}

	case "CNAME":
		return libdns.CNAME{
			Name:   name,
			TTL:    ttl,
			Target: item.Data,
		}

	case "MX":
		return libdns.MX{
			Name:       name,
			TTL:        ttl,
			Preference: uint16(item.Priority),
			Target:     item.Data,
		}

	case "NS":
		return libdns.NS{
			Name:   name,
			TTL:    ttl,
			Target: item.Data,
		}

	case "SRV":
		return parseSRVRecord(name, ttl, item)

	case "CAA":
		return parseCAARecord(name, ttl, item)

	default:
		return libdns.RR{
			Name: name,
			TTL:  ttl,
			Type: item.RecordType,
			Data: item.Data,
		}
	}
}

// parseSRVRecord parses a Gidinet SRV record into a libdns.SRV.
// SRV Data format from Gidinet is expected to be: "priority weight port target"
// but priority is also in the Priority field. The Name for SRV includes _service._proto prefix.
func parseSRVRecord(name string, ttl time.Duration, item dnsRecordListItem) libdns.Record {
	// SRV data: "weight port target" (priority is separate in Gidinet)
	// or possibly "priority weight port target"
	parts := strings.Fields(item.Data)

	var weight, port uint16
	var target string

	if len(parts) >= 3 {
		// Try parsing as "weight port target"
		w, err1 := strconv.ParseUint(parts[0], 10, 16)
		p, err2 := strconv.ParseUint(parts[1], 10, 16)
		if err1 == nil && err2 == nil {
			weight = uint16(w)
			port = uint16(p)
			target = parts[2]
		}
	}

	// Extract service and transport from the name (e.g. "_sip._tcp.example" -> service="sip", transport="tcp")
	var service, transport, srvName string
	nameParts := strings.SplitN(name, ".", 3)
	if len(nameParts) >= 3 && strings.HasPrefix(nameParts[0], "_") && strings.HasPrefix(nameParts[1], "_") {
		service = strings.TrimPrefix(nameParts[0], "_")
		transport = strings.TrimPrefix(nameParts[1], "_")
		srvName = nameParts[2]
	} else {
		// Cannot parse service/transport, fall back to RR
		return libdns.RR{
			Name: name,
			TTL:  ttl,
			Type: "SRV",
			Data: item.Data,
		}
	}

	return libdns.SRV{
		Service:   service,
		Transport: transport,
		Name:      srvName,
		TTL:       ttl,
		Priority:  uint16(item.Priority),
		Weight:    weight,
		Port:      port,
		Target:    target,
	}
}

// parseCAARecord parses a Gidinet CAA record into a libdns.CAA.
// CAA Data format from Gidinet: 'flags tag "value"'
func parseCAARecord(name string, ttl time.Duration, item dnsRecordListItem) libdns.Record {
	// Parse: flags tag "value"
	data := strings.TrimSpace(item.Data)
	parts := strings.SplitN(data, " ", 3)
	if len(parts) < 3 {
		// Cannot parse, fall back to RR
		return libdns.RR{
			Name: name,
			TTL:  ttl,
			Type: "CAA",
			Data: item.Data,
		}
	}

	flags, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil {
		return libdns.RR{
			Name: name,
			TTL:  ttl,
			Type: "CAA",
			Data: item.Data,
		}
	}

	tag := parts[1]
	value := parts[2]
	// Strip surrounding quotes from value
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}

	return libdns.CAA{
		Name:  name,
		TTL:   ttl,
		Flags: uint8(flags),
		Tag:   tag,
		Value: value,
	}
}

// recordToGidinet converts a libdns.Record to a Gidinet dnsRecord for API calls.
func (p *Provider) recordToGidinet(rec libdns.Record, domain string) (dnsRecord, error) {
	rr := rec.RR()

	ttl, err := normalizeTTL(rr.TTL)
	if err != nil {
		return dnsRecord{}, err
	}

	var priority uint
	data := rr.Data

	// Handle specific types for priority and data formatting
	switch r := rec.(type) {
	case libdns.MX:
		priority = uint(r.Preference)
		data = r.Target
	case libdns.TXT:
		data = r.Text
	default:
		// For other types, use the RR data as-is
	}

	return dnsRecord{
		DomainName: domain,
		HostName:   relativeToFQDN(rr.Name, domain),
		RecordType: rr.Type,
		Data:       data,
		TTL:        ttl,
		Priority:   priority,
	}, nil
}

// GetRecords returns all DNS records for the given zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	domain := zoneToDomain(zone)
	p.log("GetRecords: zone=%q domain=%q", zone, domain)

	client := p.getClient()
	result, err := client.recordGetList(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("GetRecords: %w", err)
	}

	if err := checkResult(&apiResult{
		ResultText:    result.ResultText,
		ResultCode:    result.ResultCode,
		ResultSubCode: result.ResultSubCode,
	}, "recordGetList"); err != nil {
		return nil, fmt.Errorf("GetRecords: %w", err)
	}

	var records []libdns.Record
	for _, item := range result.ResultItems {
		rec := toLibdnsRecord(item, domain)
		records = append(records, rec)
		rr := rec.RR()
		p.log("GetRecords: found record type=%s name=%q data=%q ttl=%s", rr.Type, rr.Name, rr.Data, rr.TTL)
	}

	p.log("GetRecords: total %d records for domain %q", len(records), domain)
	return records, nil
}

// AppendRecords adds the given DNS records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	domain := zoneToDomain(zone)
	p.log("AppendRecords: zone=%q domain=%q count=%d", zone, domain, len(recs))

	client := p.getClient()
	var added []libdns.Record

	for _, rec := range recs {
		gidiRec, err := p.recordToGidinet(rec, domain)
		if err != nil {
			return added, fmt.Errorf("AppendRecords: failed to convert record: %w", err)
		}

		p.log("AppendRecords: adding type=%s hostname=%q data=%q ttl=%d", gidiRec.RecordType, gidiRec.HostName, gidiRec.Data, gidiRec.TTL)

		result, err := client.recordAdd(ctx, gidiRec)
		if err != nil {
			return added, fmt.Errorf("AppendRecords: %w", err)
		}

		if err := checkResult(result, "recordAdd"); err != nil {
			return added, fmt.Errorf("AppendRecords: %w", err)
		}

		added = append(added, rec)
		p.log("AppendRecords: successfully added record type=%s hostname=%q", gidiRec.RecordType, gidiRec.HostName)
	}

	return added, nil
}

// SetRecords sets the given DNS records in the zone, creating or updating as needed.
// For each (name, type) pair in the input, it ensures that only the provided records
// exist in the zone for that pair, removing any extras.
func (p *Provider) SetRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	domain := zoneToDomain(zone)
	p.log("SetRecords: zone=%q domain=%q count=%d", zone, domain, len(recs))

	client := p.getClient()

	// Fetch existing records
	listResult, err := client.recordGetList(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("SetRecords: failed to get existing records: %w", err)
	}
	if err := checkResult(&apiResult{
		ResultText:    listResult.ResultText,
		ResultCode:    listResult.ResultCode,
		ResultSubCode: listResult.ResultSubCode,
	}, "recordGetList"); err != nil {
		return nil, fmt.Errorf("SetRecords: %w", err)
	}

	// Build a set of (name, type) pairs from the input
	type nameType struct {
		name     string
		recType  string
	}
	inputPairs := make(map[nameType]bool)
	for _, rec := range recs {
		rr := rec.RR()
		inputPairs[nameType{name: rr.Name, recType: rr.Type}] = true
	}

	// Delete existing records that match any (name, type) pair in the input
	for _, existing := range listResult.ResultItems {
		relName := fqdnToRelative(existing.HostName, domain)
		nt := nameType{name: relName, recType: existing.RecordType}
		if inputPairs[nt] {
			if existing.ReadOnly {
				p.log("SetRecords: skipping read-only record type=%s hostname=%q", existing.RecordType, existing.HostName)
				continue
			}

			delRec := dnsRecord{
				DomainName: domain,
				HostName:   existing.HostName,
				RecordType: existing.RecordType,
				Data:       existing.Data,
				TTL:        uint(existing.TTL),
				Priority:   uint(existing.Priority),
			}

			p.log("SetRecords: deleting old record type=%s hostname=%q data=%q", delRec.RecordType, delRec.HostName, delRec.Data)

			delResult, err := client.recordDelete(ctx, delRec)
			if err != nil {
				return nil, fmt.Errorf("SetRecords: failed to delete old record: %w", err)
			}
			if err := checkResult(delResult, "recordDelete"); err != nil {
				return nil, fmt.Errorf("SetRecords: %w", err)
			}
		}
	}

	// Add the new records
	var set []libdns.Record
	for _, rec := range recs {
		gidiRec, err := p.recordToGidinet(rec, domain)
		if err != nil {
			return set, fmt.Errorf("SetRecords: failed to convert record: %w", err)
		}

		p.log("SetRecords: adding record type=%s hostname=%q data=%q ttl=%d", gidiRec.RecordType, gidiRec.HostName, gidiRec.Data, gidiRec.TTL)

		addResult, err := client.recordAdd(ctx, gidiRec)
		if err != nil {
			return set, fmt.Errorf("SetRecords: recordAdd failed: %w", err)
		}
		if err := checkResult(addResult, "recordAdd"); err != nil {
			return set, fmt.Errorf("SetRecords: %w", err)
		}

		set = append(set, rec)
		p.log("SetRecords: successfully set record type=%s hostname=%q", gidiRec.RecordType, gidiRec.HostName)
	}

	return set, nil
}

// DeleteRecords deletes the given DNS records from the zone.
// It returns only the records that were actually deleted.
// Records that do not exist on the server are silently ignored.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	domain := zoneToDomain(zone)
	p.log("DeleteRecords: zone=%q domain=%q count=%d", zone, domain, len(recs))

	client := p.getClient()

	// Fetch existing records to get exact TTL/Priority values needed for deletion
	listResult, err := client.recordGetList(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("DeleteRecords: failed to get existing records: %w", err)
	}
	if err := checkResult(&apiResult{
		ResultText:    listResult.ResultText,
		ResultCode:    listResult.ResultCode,
		ResultSubCode: listResult.ResultSubCode,
	}, "recordGetList"); err != nil {
		return nil, fmt.Errorf("DeleteRecords: %w", err)
	}

	var deleted []libdns.Record

	for _, rec := range recs {
		rr := rec.RR()
		hostName := relativeToFQDN(rr.Name, domain)

		// Determine the data value to match
		var matchData string
		switch r := rec.(type) {
		case libdns.TXT:
			matchData = r.Text
		case libdns.MX:
			matchData = r.Target
		default:
			matchData = rr.Data
		}

		// Find matching records on the server
		found := false
		for _, existing := range listResult.ResultItems {
			if existing.ReadOnly {
				continue
			}

			// Match by hostname
			if !strings.EqualFold(existing.HostName, hostName) {
				continue
			}

			// Match by type (if specified)
			if rr.Type != "" && !strings.EqualFold(existing.RecordType, rr.Type) {
				continue
			}

			// Match by data (if specified)
			if matchData != "" && existing.Data != matchData {
				continue
			}

			// Match by TTL (if specified)
			if rr.TTL > 0 && int(rr.TTL.Seconds()) != existing.TTL {
				continue
			}

			delRec := dnsRecord{
				DomainName: domain,
				HostName:   existing.HostName,
				RecordType: existing.RecordType,
				Data:       existing.Data,
				TTL:        uint(existing.TTL),
				Priority:   uint(existing.Priority),
			}

			p.log("DeleteRecords: deleting type=%s hostname=%q data=%q ttl=%d", delRec.RecordType, delRec.HostName, delRec.Data, delRec.TTL)

			delResult, err := client.recordDelete(ctx, delRec)
			if err != nil {
				return deleted, fmt.Errorf("DeleteRecords: %w", err)
			}

			if err := checkResult(delResult, "recordDelete"); err != nil {
				return deleted, fmt.Errorf("DeleteRecords: %w", err)
			}

			deleted = append(deleted, toLibdnsRecord(existing, domain))
			found = true
			p.log("DeleteRecords: successfully deleted record type=%s hostname=%q", existing.RecordType, existing.HostName)
		}

		if !found {
			p.log("DeleteRecords: record type=%s name=%q not found on server, skipping", rr.Type, rr.Name)
		}
	}

	return deleted, nil
}

// Interface guards — ensure Provider implements the required libdns interfaces.
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
