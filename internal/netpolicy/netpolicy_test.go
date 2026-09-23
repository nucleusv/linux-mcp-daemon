package netpolicy

import (
	"context"
	"net"
	"strings"
	"testing"
)

func TestNilPolicyAllowsEverything(t *testing.T) {
	var p *Policy
	for _, ip := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "8.8.8.8"} {
		if err := p.Check(ip, net.ParseIP(ip)); err != nil {
			t.Errorf("nil policy blocked %s: %v", ip, err)
		}
	}
}

func TestDenyPrivate(t *testing.T) {
	p := &Policy{DenyPrivate: true}
	blocked := []string{
		"127.0.0.1", "127.8.9.1", "::1", "0.0.0.0", "::",
		"10.1.2.3", "172.16.0.1", "172.31.255.255", "192.168.1.1",
		"169.254.169.254", "fe80::1", "fd00::1", "100.64.0.1",
		"::ffff:127.0.0.1", "::ffff:10.0.0.1", "224.0.0.1",
	}
	for _, s := range blocked {
		if err := p.Check(s, net.ParseIP(s)); err == nil {
			t.Errorf("deny_private let %s through", s)
		}
	}
	for _, s := range []string{"8.8.8.8", "1.1.1.1", "172.32.0.1", "2606:4700::1111", "100.128.0.1"} {
		if err := p.Check(s, net.ParseIP(s)); err != nil {
			t.Errorf("deny_private blocked public %s: %v", s, err)
		}
	}
}

func TestAllowAndDenyLists(t *testing.T) {
	p := &Policy{
		DenyPrivate: true,
		Allow:       []string{"10.0.5.0/24", "intranet.example.com", "*.corp.example.com", "192.168.1.10"},
		Deny:        []string{"10.0.5.66", "*.bad.example.com"},
	}
	cases := []struct {
		host, ip string
		ok       bool
	}{
		{"10.0.5.7", "10.0.5.7", true},              // allowed CIDR
		{"10.0.6.7", "10.0.6.7", false},             // private, not in allow
		{"10.0.5.66", "10.0.5.66", false},           // deny beats allow
		{"intranet.example.com", "10.9.9.9", true},  // allowed hostname
		{"a.b.corp.example.com", "10.9.9.9", true},  // wildcard subdomain
		{"corp.example.com", "10.9.9.9", false},     // wildcard doesn't match the apex
		{"x.bad.example.com", "8.8.8.8", false},     // denied even if public
		{"192.168.1.10", "192.168.1.10", true},      // single allowed IP
		{"INTRANET.EXAMPLE.COM.", "10.9.9.9", true}, // case/trailing dot
		{"evil.example.com", "127.0.0.1", false},    // public name resolving to loopback
	}
	for _, c := range cases {
		err := p.Check(c.host, net.ParseIP(c.ip))
		if (err == nil) != c.ok {
			t.Errorf("Check(%s, %s) = %v, want ok=%v", c.host, c.ip, err, c.ok)
		}
	}
}

func TestValidate(t *testing.T) {
	good := &Policy{Allow: []string{"10.0.0.0/8", "1.2.3.4", "::1", "host.example", "*.example.com"}}
	if err := good.Validate(); err != nil {
		t.Errorf("valid policy rejected: %v", err)
	}
	for _, bad := range []string{"10.0.0.0/33", "a/b", "", "*foo"} {
		if err := (&Policy{Allow: []string{bad}}).Validate(); err == nil {
			t.Errorf("invalid rule %q accepted", bad)
		}
	}
}

func TestDialBlocksLoopbackListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	_, err = (&Policy{DenyPrivate: true}).DialContext(context.Background(), "tcp", ln.Addr().String())
	if err == nil || !strings.Contains(err.Error(), "deny_private") {
		t.Errorf("dial to loopback not blocked: %v", err)
	}
	// "localhost" must be resolved and checked too, not passed through by name.
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	if _, err := (&Policy{DenyPrivate: true}).DialContext(context.Background(), "tcp", "localhost:"+port); err == nil {
		t.Error("dial to localhost by name not blocked")
	}
	// No policy: unchanged behavior.
	c, err := (*Policy)(nil).DialContext(context.Background(), "tcp", ln.Addr().String())
	if err != nil {
		t.Errorf("nil policy failed to dial: %v", err)
	} else {
		c.Close()
	}
	// Allowlisted loopback works.
	c, err = (&Policy{DenyPrivate: true, Allow: []string{"127.0.0.0/8"}}).DialContext(context.Background(), "tcp", ln.Addr().String())
	if err != nil {
		t.Errorf("allowlisted loopback blocked: %v", err)
	} else {
		c.Close()
	}
}
