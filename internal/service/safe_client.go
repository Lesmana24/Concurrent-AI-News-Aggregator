package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// NewSafeHTTPClient creates a production-grade hardened HTTP client that completely eliminates
// SSRF Time-of-Check to Time-of-Use (TOCTOU) and DNS Rebinding vulnerabilities.
//
// Security guarantees:
// 1. Socket-Level Pinning: Resolves DNS inside DialContext and forces the TCP connection
//    directly to a verified public IP, preventing race condition DNS rebinding.
// 2. Strict Redirect Validation: Re-evaluates every HTTP 3xx redirect against scheme and IP restrictions.
// 3. Dynamic Test Bypass: Dynamically honors ALLOW_INTERNAL_URLS=true per-request for internal test servers.
func NewSafeHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid address '%s': %w", addr, err)
			}

			// Block connection to dangerous non-HTTP/management ports
			if isDangerousPort(port) {
				return nil, fmt.Errorf("connection to dangerous port '%s' is blocked (anti-protocol smuggling)", port)
			}

			allowInternal := os.Getenv("ALLOW_INTERNAL_URLS") == "true"
			if allowInternal {
				return dialer.DialContext(ctx, network, addr)
			}

			// If connecting directly to an IP literal
			if ip := net.ParseIP(host); ip != nil {
				if isRestrictedIP(ip) {
					return nil, fmt.Errorf("connection to restricted IP '%s' is blocked (SSRF defense)", ip.String())
				}
				return dialer.DialContext(ctx, network, addr)
			}

			// Hostname provided: check blacklist first
			if isRestrictedHost(host) {
				return nil, fmt.Errorf("connection to restricted host '%s' is blocked (SSRF defense)", host)
			}

			// Resolve host IP addresses right before socket creation
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, fmt.Errorf("DNS resolution failed for '%s': %w", host, err)
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("no IP address found for host '%s'", host)
			}

			// Search for the first safe public IP
			var targetIP net.IP
			for _, ip := range ips {
				if !isRestrictedIP(ip) {
					targetIP = ip
					break
				}
			}

			if targetIP == nil {
				return nil, fmt.Errorf("all resolved IP addresses for '%s' are restricted/internal (DNS rebinding blocked)", host)
			}

			// CRITICAL TOCTOU MITIGATION:
			// Connect strictly to the verified public targetIP, completely bypassing any rebinding!
			targetAddr := net.JoinHostPort(targetIP.String(), port)
			return dialer.DialContext(ctx, network, targetAddr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		MaxConnsPerHost:       50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("stopped after 5 redirects")
			}

			// Cross-origin header stripping to prevent credential leakage
			if len(via) > 0 {
				initialOrigin := strings.ToLower(via[0].URL.Host)
				if initialOrigin == "" {
					initialOrigin = strings.ToLower(via[0].URL.Hostname())
				}
				currentOrigin := strings.ToLower(req.URL.Host)
				if currentOrigin == "" {
					currentOrigin = strings.ToLower(req.URL.Hostname())
				}

				if currentOrigin != initialOrigin {
					req.Header.Del("Authorization")
					req.Header.Del("X-API-Key")
					req.Header.Del("Cookie")
					req.Header.Del("Proxy-Authorization")
				}
			}

			scheme := strings.ToLower(req.URL.Scheme)
			if scheme != "http" && scheme != "https" {
				return fmt.Errorf("redirect to unsupported scheme '%s' blocked", scheme)
			}

			port := req.URL.Port()
			if port != "" && isDangerousPort(port) {
				return fmt.Errorf("redirect to dangerous port '%s' blocked (anti-protocol smuggling)", port)
			}

			allowInternal := os.Getenv("ALLOW_INTERNAL_URLS") == "true"
			if allowInternal {
				return nil
			}

			host := strings.ToLower(req.URL.Hostname())
			if isRestrictedHost(host) {
				return fmt.Errorf("redirect to restricted host '%s' blocked", host)
			}

			// Validate IP addresses (supports direct IP literals and resolves hostnames)
			if err := validateHostIP(host); err != nil {
				return fmt.Errorf("redirect blocked by security policy: %w", err)
			}

			return nil
		},
	}

	return client
}

// isDangerousPort detects commonly targeted non-HTTP ports to prevent protocol smuggling
// and internal port scanning (e.g. Redis, Postgres, SSH, Docker, SMTP).
func isDangerousPort(port string) bool {
	switch port {
	case "21", // FTP
		"22",    // SSH
		"23",    // Telnet
		"25",    // SMTP
		"53",    // DNS
		"110",   // POP3
		"143",   // IMAP
		"445",   // SMB
		"2375",  // Docker plaintext
		"2376",  // Docker TLS
		"2379",  // Etcd client
		"2380",  // Etcd peer
		"3306",  // MySQL
		"5432",  // PostgreSQL
		"6379",  // Redis
		"11211", // Memcached
		"27017": // MongoDB
		return true
	default:
		return false
	}
}

