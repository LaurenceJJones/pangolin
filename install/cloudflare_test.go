package main

import "testing"

func TestNormalizePublicIP(t *testing.T) {
	t.Parallel()
	got, ok := normalizePublicIP("  203.0.113.4  ")
	if !ok || got != "203.0.113.4" {
		t.Fatalf("ipv4: ok=%v got=%q", ok, got)
	}
	_, ok = normalizePublicIP("10.0.0.1")
	if ok {
		t.Fatal("expected private IPv4 to be rejected")
	}
	_, ok = normalizePublicIP("pangolin.example.com")
	if ok {
		t.Fatal("expected hostname to be rejected")
	}
	got6, ok := normalizePublicIP("[2001:db8::1]")
	if !ok {
		t.Fatal("expected documentation IPv6 to parse (global unicast)")
	}
	if got6 == "" {
		t.Fatal("empty ipv6 string")
	}
}
