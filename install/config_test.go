package main

import (
	"testing"
)

func TestParseTraefikWebTrustedIPs(t *testing.T) {
	yamlDoc := `entryPoints:
  web:
    address: ":80"
    forwardedHeaders:
      trustedIPs: &cloudflare_trusted_ips
        - "173.245.48.0/20"
        - "2400:cb00::/32"
  websecure:
    address: ":443"
    forwardedHeaders:
      trustedIPs: *cloudflare_trusted_ips
`
	ips, ok := parseTraefikWebTrustedIPs([]byte(yamlDoc))
	if !ok {
		t.Fatal("expected trusted IPs to be detected")
	}
	if len(ips) != 2 {
		t.Fatalf("got %d IPs, want 2: %v", len(ips), ips)
	}
	if ips[0] != "173.245.48.0/20" || ips[1] != "2400:cb00::/32" {
		t.Fatalf("unexpected order or values: %#v", ips)
	}
}

func TestParseTraefikWebTrustedIPs_missing(t *testing.T) {
	yamlDoc := `entryPoints:
  web:
    address: ":80"
`
	_, ok := parseTraefikWebTrustedIPs([]byte(yamlDoc))
	if ok {
		t.Fatal("expected no trusted IPs")
	}
}
