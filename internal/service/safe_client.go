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
			allowInternal := os.Getenv("ALLOW_INTERNAL_URLS") == "true"
			if allowInternal {
				return dialer.DialContext(ctx, network, addr)
			}

			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid address '%s': %w", addr, err)
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
		MaxIdleConnsPerHost:   10,
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

			allowInternal := os.Getenv("ALLOW_INTERNAL_URLS") == "true"
			if allowInternal {
				return nil
			}

			scheme := strings.ToLower(req.URL.Scheme)
			if scheme != "http" && scheme != "https" {
				return fmt.Errorf("redirect to unsupported scheme '%s' blocked", scheme)
			}

			host := strings.ToLower(req.URL.Hostname())
			if isRestrictedHost(host) {
				return fmt.Errorf("redirect to restricted host '%s' blocked", host)
			}

			if ip := net.ParseIP(host); ip != nil {
				if isRestrictedIP(ip) {
					return fmt.Errorf("redirect to restricted IP '%s' blocked", ip.String())
				}
			}

			return nil
		},
	}

	return client
}

