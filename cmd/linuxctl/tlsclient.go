package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/nucleusv/linux-mcp-daemon/internal/tlsutil"
)

var (
	caCertFlag      = flag.String("ca-cert", "", "Trust this certificate (PEM) for the mcpd server - e.g. its self-signed cert (default from MCP_CA_CERT)")
	fingerprintFlag = flag.String("tls-fingerprint", "", "Trust the mcpd server whose certificate has this SHA-256 fingerprint (default from MCP_TLS_FINGERPRINT; mcpd logs it at startup)")
	insecureFlag    = flag.Bool("insecure", false, "Skip TLS certificate verification (or MCP_INSECURE=1) - for testing only")
)

// localCertPaths are where a self-signed mcpd certificate is found on the
// daemon's own host, trusted automatically when readable.
func localCertPaths() []string {
	var paths []string
	if dir := os.Getenv("MCPD_CONFIG_DIR"); dir != "" {
		paths = append(paths, filepath.Join(dir, "tls", "mcpd.crt"))
	}
	return append(paths, "/etc/mcpd/configs/tls/mcpd.crt")
}

var (
	clientOnce sync.Once
	client     *http.Client
	clientErr  error
)

// httpClient returns the client every request to mcpd goes through. Trust,
// in order: a pinned fingerprint (MCP_TLS_FINGERPRINT), a given
// certificate (MCP_CA_CERT), the system's roots plus the local mcpd
// certificate when readable; or nothing at all with MCP_INSECURE=1.
func httpClient() (*http.Client, error) {
	clientOnce.Do(func() {
		cfg := &tls.Config{MinVersion: tls.VersionTLS12}
		fp := firstNonEmpty(*fingerprintFlag, os.Getenv("MCP_TLS_FINGERPRINT"))
		ca := firstNonEmpty(*caCertFlag, os.Getenv("MCP_CA_CERT"))
		switch {
		case *insecureFlag || os.Getenv("MCP_INSECURE") == "1":
			fmt.Fprintln(os.Stderr, "Warning: TLS certificate verification is off (--insecure / MCP_INSECURE)")
			cfg.InsecureSkipVerify = true
		case fp != "":
			want, err := tlsutil.NormalizeFingerprint(fp)
			if err != nil {
				clientErr = err
				return
			}
			// Pinning replaces chain validation: the server is trusted
			// exactly when its certificate is the one pinned.
			cfg.InsecureSkipVerify = true
			cfg.VerifyConnection = func(cs tls.ConnectionState) error {
				if len(cs.PeerCertificates) == 0 {
					return errors.New("server sent no certificate")
				}
				if got := tlsutil.Fingerprint(cs.PeerCertificates[0].Raw); got != want {
					return fmt.Errorf("server certificate fingerprint %s does not match MCP_TLS_FINGERPRINT %s", got, want)
				}
				return nil
			}
		default:
			pool, err := x509.SystemCertPool()
			if err != nil || pool == nil {
				pool = x509.NewCertPool()
			}
			files := localCertPaths()
			if ca != "" {
				files = []string{ca}
			}
			for _, f := range files {
				pem, err := os.ReadFile(f)
				if err != nil {
					if ca != "" {
						clientErr = fmt.Errorf("MCP_CA_CERT: %v", err)
						return
					}
					continue // local cert absent or not readable: fine
				}
				if !pool.AppendCertsFromPEM(pem) && ca != "" {
					clientErr = fmt.Errorf("MCP_CA_CERT %s: no PEM certificate", f)
					return
				}
			}
			cfg.RootCAs = pool
		}
		client = &http.Client{Transport: &http.Transport{TLSClientConfig: cfg, Proxy: http.ProxyFromEnvironment}}
	})
	return client, clientErr
}

// tlsHint explains a certificate error in terms of what to set.
func tlsHint(err error) string {
	var unknown x509.UnknownAuthorityError
	var verify *tls.CertificateVerificationError
	var hostname x509.HostnameError
	switch {
	case errors.As(err, &unknown), errors.As(err, &verify), errors.As(err, &hostname):
		return "\nThe server's TLS certificate isn't trusted. mcpd's default one is self-signed; trust it with\n" +
			"  MCP_TLS_FINGERPRINT=sha256:...   (logged by mcpd at startup; `linuxctl describe mcpd tls` on its host)\n" +
			"or MCP_CA_CERT=/path/to/mcpd.crt   (the certificate file, /etc/mcpd/configs/tls/mcpd.crt on its host)"
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
