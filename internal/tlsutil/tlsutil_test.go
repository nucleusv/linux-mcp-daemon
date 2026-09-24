package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureSelfSigned(t *testing.T) {
	dir := t.TempDir()
	cert, key := filepath.Join(dir, "tls/mcpd.crt"), filepath.Join(dir, "tls/mcpd.key")
	created, err := EnsureSelfSigned(cert, key, []string{"mcp.example.com", "203.0.113.7"})
	if err != nil || !created {
		t.Fatalf("created=%t err=%v", created, err)
	}
	if info, _ := os.Stat(key); info.Mode().Perm() != 0o600 {
		t.Errorf("key mode %v", info.Mode().Perm())
	}
	pair, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		t.Fatal(err)
	}
	fp, parsed, err := FileFingerprint(cert)
	if err != nil || fp != Fingerprint(pair.Certificate[0]) {
		t.Fatalf("fingerprint %s %v", fp, err)
	}
	// Trusting the certificate itself verifies it for its names.
	pool := x509.NewCertPool()
	pool.AddCert(parsed)
	for _, name := range []string{"localhost", "mcp.example.com", "127.0.0.1", "203.0.113.7"} {
		if _, err := parsed.Verify(x509.VerifyOptions{Roots: pool, DNSName: name}); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	// A second call keeps the existing pair.
	if created, err := EnsureSelfSigned(cert, key, nil); err != nil || created {
		t.Errorf("second call: created=%t err=%v", created, err)
	}
	fp2, _, _ := FileFingerprint(cert)
	if fp2 != fp {
		t.Error("existing certificate was replaced")
	}
	// Only one of the two files: refuse rather than overwrite.
	os.Remove(key)
	if _, err := EnsureSelfSigned(cert, key, nil); err == nil || !strings.Contains(err.Error(), "only one of") {
		t.Errorf("half a pair: %v", err)
	}
}

func TestNormalizeFingerprint(t *testing.T) {
	want := "sha256:" + strings.Repeat("ab", 32)
	for _, in := range []string{want, strings.Repeat("AB", 32), "SHA256:" + strings.TrimSuffix(strings.Repeat("AB:", 32), ":")} {
		if got, err := NormalizeFingerprint(in); err != nil || got != want {
			t.Errorf("%q: %s %v", in, got, err)
		}
	}
	if _, err := NormalizeFingerprint("sha256:1234"); err == nil {
		t.Error("short fingerprint accepted")
	}
}
