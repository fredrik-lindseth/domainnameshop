package provider

import (
	"net/netip"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

func TestNumericMXAndSRVFieldsConvert(t *testing.T) {
	mx, err := toLibdns(dnsRecord{Host: "@", Type: "MX", Data: "mail.example.no", TTL: 60, Priority: 10})
	if err != nil {
		t.Fatal(err)
	}
	if got := mx.(libdns.MX); got.Preference != 10 || got.Target != "mail.example.no." {
		t.Fatalf("unexpected MX: %+v", got)
	}
	srv, err := toLibdns(dnsRecord{Host: "_sip._tcp", Type: "SRV", Data: "sip.example.no", TTL: 60, Priority: 1, Weight: 2, Port: 5060})
	if err != nil {
		t.Fatal(err)
	}
	if got := srv.(libdns.SRV); got.Priority != 1 || got.Weight != 2 || got.Port != 5060 {
		t.Fatalf("unexpected SRV: %+v", got)
	}
}

func TestRecordIdentityIncludesTypeAndMXPriority(t *testing.T) {
	if sameRecord(dnsRecord{Host: "@", Type: "TXT", Data: "token", TTL: 60}, libdns.RR{Name: "@", Type: "CNAME", Data: "token", TTL: time.Minute}, "example.no") {
		t.Fatal("different record types matched")
	}
	mx := libdns.MX{Name: "@", Target: "mail.example.no.", Preference: 20, TTL: time.Minute}
	if sameRecord(dnsRecord{Host: "@", Type: "MX", Data: "mail.example.no", TTL: 60, Priority: 10}, mx, "example.no") {
		t.Fatal("different MX priorities matched")
	}
}

func TestRecordRequestSupportsGenericMX(t *testing.T) {
	record, err := recordRequest(libdns.RR{Name: "@", Type: "MX", Data: "10 mail.example.no.", TTL: time.Minute}, "example.no")
	if err != nil {
		t.Fatal(err)
	}
	if record.Priority != 10 || record.Data != "mail.example.no" {
		t.Fatalf("unexpected request: %+v", record)
	}
	address, err := recordRequest(libdns.Address{Name: "www", IP: netip.MustParseAddr("192.0.2.1"), TTL: time.Minute}, "example.no")
	if err != nil || address.Type != "A" {
		t.Fatalf("unexpected A request: %+v, %v", address, err)
	}
}
