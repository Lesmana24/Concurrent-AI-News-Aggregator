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
		// If DNS resolution fails, allow if not an obviously restricted domain
		return nil
	}

	for _, ip := range ips {
		if isRestrictedIP(ip) {
			return fmt.Errorf("domain '%s' resolves to internal/restricted IP '%s'", host, ip.String())
		}
	}

	return nil
}

func isRestrictedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	// Check IPv4 169.254.x.x (AWS/GCP metadata)
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}
	return false
}
