package gidinet

import (
	"context"
	"encoding/base64"
	"fmt"
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
func toLibdnsRecord(item dnsRecordListItem, domain string) libdns.Record {
	return libdns.Record{
		ID:    buildRecordID(item),
		Type:  item.RecordType,
		Name:  fqdnToRelative(item.HostName, domain),
		Value: item.Data,
		TTL:   time.Duration(item.TTL) * time.Second,
	}
}

// buildRecordID creates a synthetic ID from the record fields since Gidinet does not provide record IDs.
// The ID is composed of: HostName|RecordType|Data
func buildRecordID(item dnsRecordListItem) string {
	return item.HostName + "|" + item.RecordType + "|" + item.Data
}

// parseRecordID parses a synthetic record ID back into its components.
// Returns hostname, recordType, data, ok.
func parseRecordID(id string) (string, string, string, bool) {
	parts := strings.SplitN(id, "|", 3)
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// toDNSRecord converts a libdns.Record to a Gidinet dnsRecord for API calls.
func (p *Provider) toDNSRecord(rec libdns.Record, domain string) (dnsRecord, error) {
	ttl, err := normalizeTTL(rec.TTL)
	if err != nil {
		return dnsRecord{}, err
	}

	var priority uint
	if strings.EqualFold(rec.Type, "MX") {
		// libdns does not have a Priority field; for MX records the priority
		// is typically encoded as part of the Value (e.g. "10 mail.example.com").
		// We default to 0 here; users needing custom priority should set it in the value.
		priority = 0
	}

	return dnsRecord{
		DomainName: domain,
		HostName:   relativeToFQDN(rec.Name, domain),
		RecordType: rec.Type,
		Data:       rec.Value,
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
		p.log("GetRecords: found record type=%s name=%q value=%q ttl=%s", rec.Type, rec.Name, rec.Value, rec.TTL)
	}

	p.log("GetRecords: total %d records for domain %q", len(records), domain)
	return records, nil
}

// AppendRecords adds the given DNS records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	domain := zoneToDomain(zone)
	p.log("AppendRecords: zone=%q domain=%q count=%d", zone, domain, len(records))

	client := p.getClient()
	var added []libdns.Record

	for _, rec := range records {
		dnsRec, err := p.toDNSRecord(rec, domain)
		if err != nil {
			return added, fmt.Errorf("AppendRecords: failed to convert record %q: %w", rec.Name, err)
		}

		p.log("AppendRecords: adding type=%s hostname=%q data=%q ttl=%d", dnsRec.RecordType, dnsRec.HostName, dnsRec.Data, dnsRec.TTL)

		result, err := client.recordAdd(ctx, dnsRec)
		if err != nil {
			return added, fmt.Errorf("AppendRecords: %w", err)
		}

		if err := checkResult(result, "recordAdd"); err != nil {
			return added, fmt.Errorf("AppendRecords: %w", err)
		}

		// Build the returned record with a synthetic ID
		addedRec := rec
		addedRec.ID = dnsRec.HostName + "|" + dnsRec.RecordType + "|" + dnsRec.Data
		addedRec.TTL = time.Duration(dnsRec.TTL) * time.Second
		added = append(added, addedRec)

		p.log("AppendRecords: successfully added record id=%q", addedRec.ID)
	}

	return added, nil
}

// SetRecords sets the given DNS records in the zone, creating or updating as needed.
// For records that have an ID (synthetic: HostName|Type|Data), it uses recordUpdate.
// For records without an ID, it uses recordAdd.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	domain := zoneToDomain(zone)
	p.log("SetRecords: zone=%q domain=%q count=%d", zone, domain, len(records))

	client := p.getClient()
	var set []libdns.Record

	for _, rec := range records {
		newDNSRec, err := p.toDNSRecord(rec, domain)
		if err != nil {
			return set, fmt.Errorf("SetRecords: failed to convert record %q: %w", rec.Name, err)
		}

		if rec.ID != "" {
			// We have a synthetic ID — try to update
			oldHostName, oldType, oldData, ok := parseRecordID(rec.ID)
			if !ok {
				return set, fmt.Errorf("SetRecords: invalid record ID %q", rec.ID)
			}

			p.log("SetRecords: updating record old=[%s %s %s] new=[%s %s %s]",
				oldHostName, oldType, oldData,
				newDNSRec.HostName, newDNSRec.RecordType, newDNSRec.Data)

			// To find the old TTL and Priority, we need to look up the existing record.
			// We fetch the current records to get the old TTL/Priority.
			listResult, err := client.recordGetList(ctx, domain)
			if err != nil {
				return set, fmt.Errorf("SetRecords: failed to get existing records: %w", err)
			}
			if err := checkResult(&apiResult{
				ResultText:    listResult.ResultText,
				ResultCode:    listResult.ResultCode,
				ResultSubCode: listResult.ResultSubCode,
			}, "recordGetList"); err != nil {
				return set, fmt.Errorf("SetRecords: %w", err)
			}

			var oldTTL uint
			var oldPriority uint
			found := false
			for _, item := range listResult.ResultItems {
				if item.HostName == oldHostName && item.RecordType == oldType && item.Data == oldData {
					oldTTL = uint(item.TTL)
					oldPriority = uint(item.Priority)
					found = true
					break
				}
			}

			if !found {
				// Old record not found — fall back to add
				p.log("SetRecords: old record not found, falling back to add")
				result, err := client.recordAdd(ctx, newDNSRec)
				if err != nil {
					return set, fmt.Errorf("SetRecords: recordAdd fallback failed: %w", err)
				}
				if err := checkResult(result, "recordAdd"); err != nil {
					return set, fmt.Errorf("SetRecords: recordAdd fallback: %w", err)
				}
			} else {
				oldRec := dnsOldRecord{
					DomainName: domain,
					HostName:   oldHostName,
					RecordType: oldType,
					Data:       oldData,
					TTL:        oldTTL,
					Priority:   oldPriority,
				}
				newRec := dnsNewRecord{
					DomainName: newDNSRec.DomainName,
					HostName:   newDNSRec.HostName,
					RecordType: newDNSRec.RecordType,
					Data:       newDNSRec.Data,
					TTL:        newDNSRec.TTL,
					Priority:   newDNSRec.Priority,
				}

				result, err := client.recordUpdate(ctx, oldRec, newRec)
				if err != nil {
					return set, fmt.Errorf("SetRecords: recordUpdate failed: %w", err)
				}
				if err := checkResult(result, "recordUpdate"); err != nil {
					return set, fmt.Errorf("SetRecords: %w", err)
				}
			}
		} else {
			// No ID — just add
			p.log("SetRecords: adding new record type=%s hostname=%q data=%q", newDNSRec.RecordType, newDNSRec.HostName, newDNSRec.Data)

			result, err := client.recordAdd(ctx, newDNSRec)
			if err != nil {
				return set, fmt.Errorf("SetRecords: recordAdd failed: %w", err)
			}
			if err := checkResult(result, "recordAdd"); err != nil {
				return set, fmt.Errorf("SetRecords: %w", err)
			}
		}

		setRec := rec
		setRec.ID = newDNSRec.HostName + "|" + newDNSRec.RecordType + "|" + newDNSRec.Data
		setRec.TTL = time.Duration(newDNSRec.TTL) * time.Second
		set = append(set, setRec)

		p.log("SetRecords: successfully set record id=%q", setRec.ID)
	}

	return set, nil
}

// DeleteRecords deletes the given DNS records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	domain := zoneToDomain(zone)
	p.log("DeleteRecords: zone=%q domain=%q count=%d", zone, domain, len(records))

	client := p.getClient()
	var deleted []libdns.Record

	for _, rec := range records {
		// If we have a synthetic ID, use it to identify the record precisely
		var dnsRec dnsRecord
		if rec.ID != "" {
			hostName, recordType, data, ok := parseRecordID(rec.ID)
			if !ok {
				return deleted, fmt.Errorf("DeleteRecords: invalid record ID %q", rec.ID)
			}

			// We need the TTL and Priority from the existing record to delete it.
			listResult, err := client.recordGetList(ctx, domain)
			if err != nil {
				return deleted, fmt.Errorf("DeleteRecords: failed to get existing records: %w", err)
			}
			if err := checkResult(&apiResult{
				ResultText:    listResult.ResultText,
				ResultCode:    listResult.ResultCode,
				ResultSubCode: listResult.ResultSubCode,
			}, "recordGetList"); err != nil {
				return deleted, fmt.Errorf("DeleteRecords: %w", err)
			}

			found := false
			for _, item := range listResult.ResultItems {
				if item.HostName == hostName && item.RecordType == recordType && item.Data == data {
					dnsRec = dnsRecord{
						DomainName: domain,
						HostName:   item.HostName,
						RecordType: item.RecordType,
						Data:       item.Data,
						TTL:        uint(item.TTL),
						Priority:   uint(item.Priority),
					}
					found = true
					break
				}
			}

			if !found {
				return deleted, fmt.Errorf("DeleteRecords: record with ID %q not found on server", rec.ID)
			}
		} else {
			// No ID — build the record from the libdns fields.
			// We still need to look up TTL/Priority from the server because Gidinet
			// requires an exact match of all fields to delete.
			hostName := relativeToFQDN(rec.Name, domain)

			listResult, err := client.recordGetList(ctx, domain)
			if err != nil {
				return deleted, fmt.Errorf("DeleteRecords: failed to get existing records: %w", err)
			}
			if err := checkResult(&apiResult{
				ResultText:    listResult.ResultText,
				ResultCode:    listResult.ResultCode,
				ResultSubCode: listResult.ResultSubCode,
			}, "recordGetList"); err != nil {
				return deleted, fmt.Errorf("DeleteRecords: %w", err)
			}

			found := false
			for _, item := range listResult.ResultItems {
				if item.HostName == hostName && item.RecordType == rec.Type && item.Data == rec.Value {
					dnsRec = dnsRecord{
						DomainName: domain,
						HostName:   item.HostName,
						RecordType: item.RecordType,
						Data:       item.Data,
						TTL:        uint(item.TTL),
						Priority:   uint(item.Priority),
					}
					found = true
					break
				}
			}

			if !found {
				return deleted, fmt.Errorf("DeleteRecords: record type=%s name=%q value=%q not found on server", rec.Type, rec.Name, rec.Value)
			}
		}

		p.log("DeleteRecords: deleting type=%s hostname=%q data=%q ttl=%d", dnsRec.RecordType, dnsRec.HostName, dnsRec.Data, dnsRec.TTL)

		result, err := client.recordDelete(ctx, dnsRec)
		if err != nil {
			return deleted, fmt.Errorf("DeleteRecords: %w", err)
		}

		if err := checkResult(result, "recordDelete"); err != nil {
			return deleted, fmt.Errorf("DeleteRecords: %w", err)
		}

		deleted = append(deleted, rec)
		p.log("DeleteRecords: successfully deleted record type=%s name=%q", rec.Type, rec.Name)
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
