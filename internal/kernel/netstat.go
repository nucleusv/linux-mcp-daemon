package kernel

// /proc/net/tcp parsing: the socket tables ss(8) and netstat(8) report,
// read directly from the kernel.

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Socket is one TCP or UDP socket from /proc/net/{tcp,tcp6,udp,udp6}.
type Socket struct {
	Netid     string    `json:"netid"` // "tcp" or "udp"
	State     string    `json:"state"` // ss's spelling: LISTEN, ESTAB, TIME-WAIT, UNCONN, ...
	RecvQ     uint64    `json:"recv_q"`
	SendQ     uint64    `json:"send_q"`
	LocalAddr string    `json:"local_address"`
	LocalPort int       `json:"local_port"`
	PeerAddr  string    `json:"peer_address"`
	PeerPort  int       `json:"peer_port"`
	UID       int       `json:"uid"`
	Inode     uint64    `json:"inode"`
	Processes []Process `json:"processes,omitempty"`
	ipv6      bool
}

// Process is a process holding a socket open.
type Process struct {
	Name string `json:"name"`
	PID  int    `json:"pid"`
	FD   int    `json:"fd"`
}

// tcpStates maps the kernel's TCP state numbers (include/net/tcp_states.h)
// to ss's names.
var tcpStates = map[uint64]string{
	0x01: "ESTAB", 0x02: "SYN-SENT", 0x03: "SYN-RECV", 0x04: "FIN-WAIT-1",
	0x05: "FIN-WAIT-2", 0x06: "TIME-WAIT", 0x07: "UNCONN", 0x08: "CLOSE-WAIT",
	0x09: "LAST-ACK", 0x0A: "LISTEN", 0x0B: "CLOSING", 0x0C: "SYN-RECV",
}

// ReadSockets reads the TCP and UDP socket tables of this process's
// network namespace. procRoot is normally "/proc".
func ReadSockets(procRoot string) ([]Socket, error) {
	var all []Socket
	for _, t := range []struct {
		file  string
		netid string
		ipv6  bool
	}{
		{"tcp", "tcp", false}, {"tcp6", "tcp", true},
		{"udp", "udp", false}, {"udp6", "udp", true},
	} {
		socks, err := parseTable(filepath.Join(procRoot, "net", t.file), t.netid, t.ipv6)
		if err != nil {
			if os.IsNotExist(err) {
				continue // e.g. IPv6 disabled
			}
			return nil, err
		}
		all = append(all, socks...)
	}
	return all, nil
}

func parseTable(path, netid string, ipv6 bool) ([]Socket, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var socks []Socket
	sc := bufio.NewScanner(f)
	sc.Scan() // header
	for sc.Scan() {
		s, err := parseLine(sc.Text(), netid, ipv6)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", path, err)
		}
		socks = append(socks, s)
	}
	return socks, sc.Err()
}

// parseLine parses e.g.
//
//	0: 00000000:0016 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 12345 1 ...
func parseLine(line, netid string, ipv6 bool) (Socket, error) {
	f := strings.Fields(line)
	if len(f) < 10 {
		return Socket{}, fmt.Errorf("short line %q", line)
	}
	s := Socket{Netid: netid, ipv6: ipv6}
	var err error
	if s.LocalAddr, s.LocalPort, err = parseAddr(f[1]); err != nil {
		return s, err
	}
	if s.PeerAddr, s.PeerPort, err = parseAddr(f[2]); err != nil {
		return s, err
	}
	st, err := strconv.ParseUint(f[3], 16, 8)
	if err != nil {
		return s, err
	}
	s.State = tcpStates[st]
	if netid == "udp" && st == 0x07 {
		s.State = "UNCONN"
	}
	if s.State == "" {
		s.State = fmt.Sprintf("UNKNOWN-%02X", st)
	}
	tx, rx, _ := strings.Cut(f[4], ":")
	s.SendQ, _ = strconv.ParseUint(tx, 16, 64)
	s.RecvQ, _ = strconv.ParseUint(rx, 16, 64)
	s.UID, _ = strconv.Atoi(f[7])
	s.Inode, _ = strconv.ParseUint(f[9], 10, 64)
	return s, nil
}

// parseAddr decodes "0100007F:0035" (IPv4) or a 32-hex-digit IPv6 address:
// each 32-bit word is in host (little-endian on x86/arm) byte order.
func parseAddr(s string) (string, int, error) {
	hexIP, hexPort, ok := strings.Cut(s, ":")
	if !ok {
		return "", 0, fmt.Errorf("bad address %q", s)
	}
	raw, err := hex.DecodeString(hexIP)
	if err != nil || (len(raw) != 4 && len(raw) != 16) {
		return "", 0, fmt.Errorf("bad address %q", s)
	}
	ip := make(net.IP, len(raw))
	for i := 0; i < len(raw); i += 4 {
		binary.BigEndian.PutUint32(ip[i:], binary.LittleEndian.Uint32(raw[i:]))
	}
	port, err := strconv.ParseUint(hexPort, 16, 16)
	if err != nil {
		return "", 0, fmt.Errorf("bad port in %q", s)
	}
	return ip.String(), int(port), nil
}

// Endpoint formats an address the way `ss -n` does: "0.0.0.0:22",
// "[::]:22", and "*" for an unset peer port.
func (s Socket) Endpoint(local bool) string {
	addr, port := s.PeerAddr, s.PeerPort
	if local {
		addr, port = s.LocalAddr, s.LocalPort
	}
	p := strconv.Itoa(port)
	if port == 0 {
		p = "*"
	}
	if s.ipv6 {
		// ss prints IPv4-mapped addresses of an IPv6 socket as [::ffff:a.b.c.d].
		if ip := net.ParseIP(addr); ip != nil && ip.To4() != nil {
			addr = "::ffff:" + ip.To4().String()
		}
		return "[" + addr + "]:" + p
	}
	return addr + ":" + p
}

// SocketOwners maps socket inodes to the processes holding them, by
// reading every /proc/<pid>/fd link ("socket:[12345]"). Processes whose fd
// directory can't be read (other users' processes, when not root) are
// skipped - as with ss -p, root sees them all.
func SocketOwners(procRoot string) map[uint64][]Process {
	owners := map[uint64][]Process{}
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return owners
	}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		fdDir := filepath.Join(procRoot, e.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		name := ""
		for _, fd := range fds {
			target, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil || !strings.HasPrefix(target, "socket:[") {
				continue
			}
			inode, err := strconv.ParseUint(strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]"), 10, 64)
			if err != nil {
				continue
			}
			if name == "" {
				comm, _ := os.ReadFile(filepath.Join(procRoot, e.Name(), "comm"))
				name = strings.TrimSpace(string(comm))
			}
			fdNum, _ := strconv.Atoi(fd.Name())
			owners[inode] = append(owners[inode], Process{Name: name, PID: pid, FD: fdNum})
		}
	}
	return owners
}
