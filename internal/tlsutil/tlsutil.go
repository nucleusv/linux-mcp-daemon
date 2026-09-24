// Package tlsutil creates mcpd's self-signed TLS certificate and computes
// the fingerprint clients pin it by.
package tlsutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Validity of a generated certificate. Clients pin it (or its file), so a
// short lifetime would only mean outages at renewal; replace it any time
// by deleting both files and restarting mcpd.
const Validity = 5 * 365 * 24 * time.Hour

// EnsureSelfSigned creates a self-signed certificate and key at certPath
// and keyPath unless both exist. It reports whether it created them. One
// file without the other is an error: overwriting either could lose a
// certificate someone installed on purpose.
func EnsureSelfSigned(certPath, keyPath string, extraHosts []string) (bool, error) {
	certOK, keyOK := exists(certPath), exists(keyPath)
	switch {
	case certOK && keyOK:
		return false, nil
	case certOK != keyOK:
		return false, fmt.Errorf("only one of %s and %s exists - provide both, or remove it to have a new pair generated", certPath, keyPath)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return false, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return false, err
	}
	dnsNames, ips := Hosts(extraHosts)
	host, _ := os.Hostname()
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "mcpd " + host, Organization: []string{"linux-mcp-daemon (self-signed)"}},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(Validity),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		// Its own CA, so it can be added as a trust anchor as is
		// (MCP_CA_CERT, NODE_EXTRA_CA_CERTS, a system trust store).
		IsCA:                  true,
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return false, err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return false, err
	}

	for _, dir := range []string{filepath.Dir(certPath), filepath.Dir(keyPath)} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return false, err
		}
	}
	// The key first, and readable by its owner only.
	if err := writeNew(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		return false, err
	}
	if err := writeNew(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		os.Remove(keyPath)
		return false, err
	}
	return true, nil
}

// Hosts lists the names and addresses a generated certificate covers: the
// host name, localhost, loopback, every address on the machine's
// interfaces, and any extra ones (tls.hosts in daemon.yaml).
func Hosts(extra []string) ([]string, []net.IP) {
	dns := map[string]bool{"localhost": true}
	if h, err := os.Hostname(); err == nil && h != "" {
		dns[h] = true
	}
	ipSet := map[string]net.IP{}
	add := func(ip net.IP) { ipSet[ip.String()] = ip }
	add(net.ParseIP("127.0.0.1"))
	add(net.ParseIP("::1"))
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && !n.IP.IsLinkLocalUnicast() {
				add(n.IP)
			}
		}
	}
	for _, h := range extra {
		if ip := net.ParseIP(h); ip != nil {
			add(ip)
		} else if h != "" {
			dns[h] = true
		}
	}
	var names []string
	for n := range dns {
		names = append(names, n)
	}
	var ips []net.IP
	for _, ip := range ipSet {
		ips = append(ips, ip)
	}
	return names, ips
}

// Fingerprint returns the SHA-256 of a certificate's DER bytes as
// "sha256:<hex>", the form MCP_TLS_FINGERPRINT takes.
func Fingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// FileFingerprint reads a PEM certificate file and returns its
// fingerprint and parsed certificate.
func FileFingerprint(certPath string) (string, *x509.Certificate, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return "", nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", nil, fmt.Errorf("%s: no PEM certificate", certPath)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", nil, err
	}
	return Fingerprint(block.Bytes), cert, nil
}

// NormalizeFingerprint accepts "sha256:ab12...", "AB:12:...", with or
// without the prefix and colons, and returns the canonical form.
func NormalizeFingerprint(fp string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(fp))
	s = strings.TrimPrefix(s, "sha256:")
	s = strings.ReplaceAll(s, ":", "")
	if b, err := hex.DecodeString(s); err != nil || len(b) != sha256.Size {
		return "", fmt.Errorf("invalid SHA-256 fingerprint %q", fp)
	}
	return "sha256:" + s, nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

func writeNew(path string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	return f.Close()
}
