// Package netpolicy restricts which destinations the outbound network tools
// (network/curl, network/ping) may connect to, per user, from the
// `network:` block in mcp-sudo.yaml.
//
// Without a policy nothing is restricted - the default is unchanged. With
// one, every connection is checked against the resolved IP address at dial
// time, and the dial goes to exactly that checked IP. Checking the hostname
// alone would be bypassable by DNS rebinding (a name that resolves to a
// public IP when checked and 127.0.0.1 when connected); dialing the checked
// IP closes that gap. HTTP redirects go through the same dialer, so each
// hop is checked too.
package netpolicy

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// Policy is one user's `network:` block.
type Policy struct {
	// DenyPrivate blocks loopback, private (RFC 1918 / ULA), link-local
	// (incl. the 169.254.169.254 cloud metadata endpoint), CGNAT,
	// unspecified and multicast addresses.
	DenyPrivate bool `yaml:"deny_private" json:"deny_private"`
	// Allow lists CIDRs, IPs or hostnames ("host.example.com", or
	// "*.example.com" for subdomains) permitted even when DenyPrivate
	// would block them.
	Allow []string `yaml:"allow,omitempty" json:"allow,omitempty"`
	// Deny lists CIDRs, IPs or hostnames that are always blocked; it wins
	// over Allow.
	Deny []string `yaml:"deny,omitempty" json:"deny,omitempty"`
}

// Validate checks every rule parses, so a typo fails at daemon startup
// rather than silently never matching.
func (p *Policy) Validate() error {
	for _, list := range [][]string{p.Allow, p.Deny} {
		for _, r := range list {
			if _, err := parseRule(r); err != nil {
				return err
			}
		}
	}
	return nil
}

type rule struct {
	cidr *net.IPNet // IP or CIDR rule
	host string     // exact hostname, lowercase
	wild string     // "*.example.com" stored as ".example.com"
}

func parseRule(r string) (rule, error) {
	r = strings.ToLower(strings.TrimSpace(r))
	if r == "" {
		return rule{}, fmt.Errorf("empty network rule")
	}
	if _, cidr, err := net.ParseCIDR(r); err == nil {
		return rule{cidr: cidr}, nil
	}
	if ip := net.ParseIP(r); ip != nil {
		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		return rule{cidr: &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}}, nil
	}
	if strings.HasPrefix(r, "*.") {
		return rule{wild: r[1:]}, nil
	}
	if strings.ContainsAny(r, "/* ") {
		return rule{}, fmt.Errorf("invalid network rule %q: expected a CIDR, IP, hostname or *.domain", r)
	}
	return rule{host: strings.TrimSuffix(r, ".")}, nil
}

func matches(rules []string, host string, ip net.IP) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	for _, s := range rules {
		r, err := parseRule(s)
		if err != nil {
			continue
		}
		switch {
		case r.cidr != nil:
			if ip != nil && r.cidr.Contains(ip) {
				return true
			}
		case r.wild != "":
			if strings.HasSuffix(host, r.wild) {
				return true
			}
		case r.host != "":
			if host == r.host {
				return true
			}
		}
	}
	return false
}

var cgnat = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

// IsPrivate reports whether ip is an address that isn't a public internet
// destination: loopback, RFC 1918/ULA, link-local, CGNAT, 0.0.0.0/::
// (which connects to the local host), or multicast.
func IsPrivate(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil {
		ip = v4 // also normalizes IPv4-mapped IPv6 (::ffff:127.0.0.1)
		if v4[0] == 0 {
			return true // 0.0.0.0/8
		}
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || cgnat.Contains(ip)
}

// Check decides whether connecting to ip (reached via hostname host, which
// may be the IP literal itself) is permitted. A nil policy permits all.
func (p *Policy) Check(host string, ip net.IP) error {
	if p == nil {
		return nil
	}
	if matches(p.Deny, host, ip) {
		return fmt.Errorf("destination %s (%s) is blocked by this user's network policy (deny list)", host, ip)
	}
	if matches(p.Allow, host, ip) {
		return nil
	}
	if p.DenyPrivate && IsPrivate(ip) {
		return fmt.Errorf("destination %s (%s) is a private/internal address, blocked by this user's network policy (deny_private)", host, ip)
	}
	return nil
}

// DialContext resolves addr, checks every candidate IP against the policy
// and connects to the first permitted one - never re-resolving, so the
// address checked is the address used. With a nil policy it's a plain dial.
func (p *Policy) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: 30 * time.Second}
	if p == nil {
		return d.DialContext(ctx, network, addr)
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	var ips []net.IP
	if ip := net.ParseIP(host); ip != nil {
		ips = []net.IP{ip}
	} else {
		addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, a := range addrs {
			ips = append(ips, a.IP)
		}
	}
	var firstErr error
	for _, ip := range ips {
		if err := p.Check(host, ip); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		conn, err := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr == nil {
		firstErr = fmt.Errorf("no addresses found for %s", host)
	}
	return nil, firstErr
}
