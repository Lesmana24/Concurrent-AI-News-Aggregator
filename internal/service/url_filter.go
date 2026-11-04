package service

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"strings"
)

var blockedExtensions = map[string]bool{
	".pdf": true, ".zip": true, ".exe": true, ".tar": true, ".gz": true,
	".rar": true, ".7z": true, ".jpg": true, ".jpeg": true, ".png": true,
	".gif": true, ".svg": true, ".mp4": true, ".mp3": true, ".wav": true,
	".avi": true, ".mov": true, ".doc": true, ".docx": true, ".xls": true,
}

// FilterURLs validates, deduplicates, and screens URLs against SSRF and non-article resources.
func FilterURLs(rawURLs []string, maxCount int) ([]string, error) {
	if maxCount <= 0 {
		maxCount = 5
	}

	seen := make(map[string]bool)
	var filtered []string

	allowInternal := os.Getenv("ALLOW_INTERNAL_URLS") == "true"

	for _, raw := range rawURLs {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}

		parsed, err := url.ParseRequestURI(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid URL '%s': %w", trimmed, err)
		}

		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "http" && scheme != "https" {
			return nil, fmt.Errorf("unsupported scheme in URL '%s': only http and https are allowed", trimmed)
		}

		host := strings.ToLower(parsed.Hostname())
		if host == "" {
			return nil, fmt.Errorf("missing host in URL '%s'", trimmed)
		}

		if !allowInternal {
			if host == "localhost" || strings.HasSuffix(host, ".localhost") {
				return nil, fmt.Errorf("access to localhost URL '%s' is prohibited", trimmed)
			}

			if isRestrictedHost(host) {
				return nil, fmt.Errorf("access to cloud metadata/restricted domain '%s' is prohibited", trimmed)
			}

			// SSRF verification: Check IP restrictions
			if err := validateHostIP(host); err != nil {
				return nil, fmt.Errorf("URL '%s' rejected: %w", trimmed, err)
			}
		}

		// File extension check
		ext := strings.ToLower(path.Ext(parsed.Path))
		if blockedExtensions[ext] {
			return nil, fmt.Errorf("URL '%s' points to a non-article file type (%s)", trimmed, ext)
		}

		// Normalize URL (strip fragment, trim trailing slash)
		parsed.Fragment = ""
		normalized := parsed.String()
		normalized = strings.TrimRight(normalized, "/")

		if !seen[normalized] {
			seen[normalized] = true
			filtered = append(filtered, normalized)
		}

		if len(filtered) >= maxCount {
			break
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no valid article URLs provided")
	}

	return filtered, nil
}

func isRestrictedHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "" {
		return true
	}
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}
	// Cloud metadata hostnames
	if h == "metadata.google.internal" || strings.HasSuffix(h, ".metadata.google.internal") ||
		h == "metadata.goog" || h == "instance-data" {
		return true
	}
	return false
}

func validateHostIP(host string) error {
	// If direct IP
	if ip := net.ParseIP(host); ip != nil {
		if isRestrictedIP(ip) {
			return fmt.Errorf("direct access to private/loopback IP '%s' is blocked", host)
		}
		return nil
	}

	// Resolve hostname
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("domain '%s' could not be resolved (unreachable or invalid domain): %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("domain '%s' has no associated IP addresses", host)
	}

	for _, ip := range ips {
		if isRestrictedIP(ip) {
			return fmt.Errorf("domain '%s' resolves to internal/restricted IP '%s'", host, ip.String())
		}
	}

	return nil
}

// isRestrictedIP inspects an IP address against loopback, private RFC 1918, link-local,
// CGNAT RFC 6598, cloud metadata, test-net, and reserved IP ranges.
func isRestrictedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	// Normalize IPv4-mapped IPv6 (e.g. ::ffff:127.0.0.1) to 4-byte IPv4 representation
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}

	if ip4 := ip.To4(); ip4 != nil {
		// 0.0.0.0/8 (Current network)
		if ip4[0] == 0 {
			return true
		}
		// 100.64.0.0/10 (RFC 6598 Shared Address Space / CGNAT / Cloud VPC - includes Alibaba 100.100.100.200)
		if ip4[0] == 100 && (ip4[1]&0xc0) == 64 {
			return true
		}
		// 169.254.0.0/16 (AWS / GCP / Azure / DigitalOcean Link-Local Metadata)
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		// 192.0.0.0/24 (IETF Protocol Assignments)
		if ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 0 {
			return true
		}
		// 192.0.2.0/24 (TEST-NET-1)
		if ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 2 {
			return true
		}
		// 198.18.0.0/15 (Network benchmark tests)
		if ip4[0] == 198 && (ip4[1] == 18 || ip4[1] == 19) {
			return true
		}
		// 198.51.100.0/24 (TEST-NET-2)
		if ip4[0] == 198 && ip4[1] == 51 && ip4[2] == 100 {
			return true
		}
		// 203.0.113.0/24 (TEST-NET-3)
		if ip4[0] == 203 && ip4[1] == 0 && ip4[2] == 113 {
			return true
		}
		// 240.0.0.0/4 (Reserved / Class E)
		if ip4[0] >= 240 {
			return true
		}
		// 255.255.255.255 (Broadcast)
		if ip4[0] == 255 && ip4[1] == 255 && ip4[2] == 255 && ip4[3] == 255 {
			return true
		}
		return false
	}

	// IPv6 Documentation (2001:db8::/32)
	if ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0x0d && ip[3] == 0xb8 {
		return true
	}

	return false
}
