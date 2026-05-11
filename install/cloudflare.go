package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Official Cloudflare edge IP list endpoints (IPv4 and IPv6).
const (
	CloudflareIPsV4URL = "https://www.cloudflare.com/ips-v4"
	CloudflareIPsV6URL = "https://www.cloudflare.com/ips-v6"
)

func fetchCloudflareIPList(client *http.Client, url string) ([]string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: status %s", url, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: empty response", url)
	}
	return out, nil
}

func fetchCloudflareIPRangesOnce() ([]string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	v4, err := fetchCloudflareIPList(client, CloudflareIPsV4URL)
	if err != nil {
		return nil, fmt.Errorf("ipv4: %w", err)
	}
	v6, err := fetchCloudflareIPList(client, CloudflareIPsV6URL)
	if err != nil {
		return nil, fmt.Errorf("ipv6: %w", err)
	}
	out := make([]string, 0, len(v4)+len(v6))
	out = append(out, v4...)
	out = append(out, v6...)
	return out, nil
}

// FetchCloudflareIPRanges returns Cloudflare's published IPv4 and IPv6 CIDR ranges
// suitable for Traefik forwardedHeaders.trustedIPs and CrowdSec forwardedHeadersTrustedIPs.
// It retries a few times on failure (transient network issues).
func FetchCloudflareIPRanges() ([]string, error) {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		ips, err := fetchCloudflareIPRangesOnce()
		if err == nil {
			return ips, nil
		}
		lastErr = err
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	return nil, lastErr
}

// normalizePublicIP trims input and returns a canonical IP string if s is a
// global unicast public address (IPv4 or IPv6), suitable for gerbil.base_endpoint.
func normalizePublicIP(s string) (string, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	s = strings.TrimSpace(s)
	ip := net.ParseIP(s)
	if ip == nil {
		return "", false
	}
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return "", false
	}
	return ip.String(), true
}
