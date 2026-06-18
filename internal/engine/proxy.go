package engine

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// BuildTransport creates an http.Transport configured for the scan.
func BuildTransport(cfg transportConfig) (*http.Transport, error) {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec
	}

	if cfg.CACert != "" {
		pem, err := os.ReadFile(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("parsing CA cert: invalid PEM")
		}
		tlsCfg.RootCAs = pool
	}

	switch strings.ToLower(cfg.TLSMinVersion) {
	case "1.0":
		tlsCfg.MinVersion = tls.VersionTLS10
	case "1.1":
		tlsCfg.MinVersion = tls.VersionTLS11
	case "1.2":
		tlsCfg.MinVersion = tls.VersionTLS12
	default:
		tlsCfg.MinVersion = tls.VersionTLS12
	}

	transport := &http.Transport{
		TLSClientConfig:       tlsCfg,
		MaxIdleConnsPerHost:   cfg.MaxConnsPerHost,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   cfg.Timeout,
		ResponseHeaderTimeout: cfg.Timeout,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
	}

	if cfg.ProxyURL != "" {
		if err := applyProxy(transport, cfg.ProxyURL); err != nil {
			return nil, err
		}
	}

	return transport, nil
}

func applyProxy(t *http.Transport, proxyURLStr string) error {
	u, err := url.Parse(proxyURLStr)
	if err != nil {
		return fmt.Errorf("parsing proxy URL: %w", err)
	}

	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		t.Proxy = http.ProxyURL(u)
	case "socks5":
		var proxyAuth *proxy.Auth
		if u.User != nil {
			proxyAuth = &proxy.Auth{User: u.User.Username()}
			if p, ok := u.User.Password(); ok {
				proxyAuth.Password = p
			}
		}
		dialer, err := proxy.SOCKS5("tcp", u.Host, proxyAuth, proxy.Direct)
		if err != nil {
			return fmt.Errorf("creating SOCKS5 dialer: %w", err)
		}
		t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
	default:
		return fmt.Errorf("unsupported proxy scheme %q", u.Scheme)
	}
	return nil
}

type transportConfig struct {
	ProxyURL           string
	CACert             string
	TLSMinVersion      string
	InsecureSkipVerify bool
	MaxConnsPerHost    int
	Timeout            time.Duration
}
