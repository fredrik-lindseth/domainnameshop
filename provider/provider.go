// Package provider implements libdns record operations for Domeneshop.
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/libdns/libdns"
)

var baseURL = "https://api.domeneshop.no/v0"

type Provider struct {
	APIToken  string `json:"api_token"`
	APISecret string `json:"api_secret"`
	mu        sync.Mutex
	client    *http.Client
	zones     map[string]zone
}

type zone struct {
	ID   int    `json:"id"`
	Name string `json:"domain"`
}
type dnsRecord struct {
	ID       int    `json:"id,omitempty"`
	Host     string `json:"host,omitempty"`
	Data     string `json:"data,omitempty"`
	Type     string `json:"type,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Weight   int    `json:"weight,omitempty"`
	Port     int    `json:"port,omitempty"`
}

func (p *Provider) httpClient() *http.Client {
	if p.client == nil {
		p.client = &http.Client{Timeout: 30 * time.Second}
	}
	return p.client
}
func (p *Provider) request(ctx context.Context, method, endpoint string, input, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(p.APIToken, p.APISecret)
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("domeneshop: %s %s: HTTP %d: %s", method, endpoint, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if output != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(output); err != nil && err != io.EOF {
			return err
		}
	}
	return nil
}
func trimZone(zone string) string { return strings.TrimSuffix(strings.ToLower(zone), ".") }
func (p *Provider) resolveZone(ctx context.Context, asked string) (zone, error) {
	key := trimZone(asked)
	if cached, ok := p.zones[key]; ok {
		return cached, nil
	}
	var zones []zone
	endpoint := baseURL + "/domains?domain=" + url.QueryEscape(key)
	if err := p.request(ctx, http.MethodGet, endpoint, nil, &zones); err != nil {
		return zone{}, err
	}
	for _, candidate := range zones {
		if strings.EqualFold(candidate.Name, key) {
			if p.zones == nil {
				p.zones = map[string]zone{}
			}
			p.zones[key] = candidate
			return candidate, nil
		}
	}
	return zone{}, fmt.Errorf("domeneshop: zone %q not found", key)
}
func (p *Provider) records(ctx context.Context, z zone) ([]dnsRecord, error) {
	var records []dnsRecord
	err := p.request(ctx, http.MethodGet, fmt.Sprintf("%s/domains/%d/dns", baseURL, z.ID), nil, &records)
	return records, err
}
func nameForAPI(name, zoneName string) string {
	name = strings.TrimSuffix(name, ".")
	zoneName = trimZone(zoneName)
	if name == "" || name == "@" || strings.EqualFold(name, zoneName) {
		return "@"
	}
	suffix := "." + zoneName
	if strings.HasSuffix(strings.ToLower(name), suffix) {
		return strings.TrimSuffix(name, name[len(name)-len(suffix):])
	}
	return name
}
func recordRequest(record libdns.Record, zoneName string) (dnsRecord, error) {
	rr := record.RR()
	result := dnsRecord{Host: nameForAPI(rr.Name, zoneName), Type: strings.ToUpper(rr.Type), Data: rr.Data, TTL: int(rr.TTL.Seconds())}
	if result.TTL == 0 {
		result.TTL = 120
	}
	switch value := record.(type) {
	case libdns.MX:
		result.Priority = int(value.Preference)
		result.Data = strings.TrimSuffix(value.Target, ".")
	case libdns.SRV:
		result.Priority = int(value.Priority)
		result.Weight = int(value.Weight)
		result.Port = int(value.Port)
		result.Data = strings.TrimSuffix(value.Target, ".")
	default:
		if result.Type == "MX" {
			fields := strings.Fields(rr.Data)
			if len(fields) != 2 {
				return result, fmt.Errorf("domeneshop: invalid MX data %q", rr.Data)
			}
			priority, err := strconv.Atoi(fields[0])
			if err != nil {
				return result, err
			}
			result.Priority = priority
			result.Data = strings.TrimSuffix(fields[1], ".")
		}
	}
	return result, nil
}
func toLibdns(record dnsRecord) (libdns.Record, error) {
	ttl := time.Duration(record.TTL) * time.Second
	switch record.Type {
	case "MX":
		return libdns.MX{Name: record.Host, TTL: ttl, Preference: uint16(record.Priority), Target: ensureDot(record.Data)}, nil
	case "SRV":
		parts := strings.SplitN(record.Host, ".", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("domeneshop: invalid SRV host %q", record.Host)
		}
		return libdns.SRV{Name: record.Host, TTL: ttl, Service: strings.TrimPrefix(parts[0], "_"), Transport: strings.TrimPrefix(parts[1], "_"), Priority: uint16(record.Priority), Weight: uint16(record.Weight), Port: uint16(record.Port), Target: ensureDot(record.Data)}, nil
	default:
		return libdns.RR{Name: record.Host, TTL: ttl, Type: record.Type, Data: record.Data}.Parse()
	}
}
func ensureDot(value string) string {
	if value != "" && !strings.HasSuffix(value, ".") {
		return value + "."
	}
	return value
}
func sameRecord(api dnsRecord, wanted libdns.Record, zoneName string) bool {
	request, err := recordRequest(wanted, zoneName)
	if err != nil {
		return false
	}
	if !strings.EqualFold(api.Host, request.Host) || !strings.EqualFold(api.Type, request.Type) || api.TTL != request.TTL {
		return false
	}
	if api.Type == "MX" {
		return api.Priority == request.Priority && strings.EqualFold(strings.TrimSuffix(api.Data, "."), strings.TrimSuffix(request.Data, "."))
	}
	if api.Type == "SRV" {
		return api.Priority == request.Priority && api.Weight == request.Weight && api.Port == request.Port && strings.EqualFold(strings.TrimSuffix(api.Data, "."), strings.TrimSuffix(request.Data, "."))
	}
	return api.Data == request.Data
}
func sameNameType(api dnsRecord, wanted libdns.Record, zoneName string) bool {
	request, err := recordRequest(wanted, zoneName)
	return err == nil && strings.EqualFold(api.Host, request.Host) && strings.EqualFold(api.Type, request.Type)
}
func (p *Provider) GetRecords(ctx context.Context, asked string) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	z, err := p.resolveZone(ctx, asked)
	if err != nil {
		return nil, err
	}
	records, err := p.records(ctx, z)
	if err != nil {
		return nil, err
	}
	out := make([]libdns.Record, 0, len(records))
	for _, record := range records {
		converted, err := toLibdns(record)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}
func (p *Provider) AppendRecords(ctx context.Context, asked string, input []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	z, err := p.resolveZone(ctx, asked)
	if err != nil {
		return nil, err
	}
	out := make([]libdns.Record, 0, len(input))
	for _, record := range input {
		request, err := recordRequest(record, z.Name)
		if err != nil {
			return out, err
		}
		var created dnsRecord
		if err = p.request(ctx, http.MethodPost, fmt.Sprintf("%s/domains/%d/dns", baseURL, z.ID), request, &created); err != nil {
			return out, err
		}
		if created.ID == 0 {
			created = request
		}
		converted, err := toLibdns(created)
		if err != nil {
			return out, err
		}
		out = append(out, converted)
	}
	return out, nil
}
func (p *Provider) DeleteRecords(ctx context.Context, asked string, input []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	z, err := p.resolveZone(ctx, asked)
	if err != nil {
		return nil, err
	}
	existing, err := p.records(ctx, z)
	if err != nil {
		return nil, err
	}
	out := []libdns.Record{}
	used := map[int]bool{}
	for _, wanted := range input {
		for _, record := range existing {
			if !used[record.ID] && sameRecord(record, wanted, z.Name) {
				if err := p.request(ctx, http.MethodDelete, fmt.Sprintf("%s/domains/%d/dns/%d", baseURL, z.ID, record.ID), nil, nil); err != nil {
					return out, err
				}
				used[record.ID] = true
				converted, _ := toLibdns(record)
				out = append(out, converted)
				break
			}
		}
	}
	return out, nil
}
func (p *Provider) SetRecords(ctx context.Context, asked string, input []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	z, err := p.resolveZone(ctx, asked)
	if err != nil {
		return nil, err
	}
	existing, err := p.records(ctx, z)
	if err != nil {
		return nil, err
	}
	groups := map[string][]libdns.Record{}
	order := []string{}
	for _, record := range input {
		request, err := recordRequest(record, z.Name)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(request.Host) + "\x00" + request.Type
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], record)
	}
	for _, key := range order {
		wanted := groups[key]
		current := []dnsRecord{}
		for _, record := range existing {
			if sameNameType(record, wanted[0], z.Name) {
				current = append(current, record)
			}
		}
		for i, record := range wanted {
			request, err := recordRequest(record, z.Name)
			if err != nil {
				return nil, err
			}
			if i < len(current) {
				if err = p.request(ctx, http.MethodPut, fmt.Sprintf("%s/domains/%d/dns/%d", baseURL, z.ID, current[i].ID), request, nil); err != nil {
					return nil, err
				}
			} else if err = p.request(ctx, http.MethodPost, fmt.Sprintf("%s/domains/%d/dns", baseURL, z.ID), request, nil); err != nil {
				return nil, err
			}
		}
		if len(current) > len(wanted) {
			for _, stale := range current[len(wanted):] {
				if err = p.request(ctx, http.MethodDelete, fmt.Sprintf("%s/domains/%d/dns/%d", baseURL, z.ID, stale.ID), nil, nil); err != nil {
					return nil, err
				}
			}
		}
	}
	final, err := p.records(ctx, z)
	if err != nil {
		return nil, err
	}
	out := []libdns.Record{}
	used := map[int]bool{}
	for _, wanted := range input {
		found := false
		for _, record := range final {
			if !used[record.ID] && sameRecord(record, wanted, z.Name) {
				converted, err := toLibdns(record)
				if err != nil {
					return out, err
				}
				used[record.ID] = true
				out = append(out, converted)
				found = true
				break
			}
		}
		if !found {
			return out, fmt.Errorf("domeneshop: record not found after set")
		}
	}
	return out, nil
}

var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
