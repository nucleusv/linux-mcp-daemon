# Network Group Implementation Plan

This document tracks the standard Linux network utilities and their implementation status in the MCP Daemon.

## Network Interfaces and Sockets

- [x] **ip address** (`network/list-interfaces`): Show / manipulate routing, network devices, interfaces and tunnels.
- [x] **ip route** (`read_routes` resource): Show routing table.
- [x] **ss / netstat** (`network/list-connections`): Utility to investigate sockets.
- [x] **ping** (`network/ping`): Send ICMP ECHO_REQUEST to network hosts.
- [ ] **traceroute / tracepath**: Print the route packets trace to network host.
- [x] **nslookup / dig** (`network/nslookup`): Query Internet name servers interactively.
- [x] **curl / wget** (`network/curl`): Transfer a URL (could be useful for MCP to verify internal endpoints).
- [x] **arp / ip neigh** (`network/arp`): Manipulate or view the system ARP cache.
- [ ] **iptables / nft**: Administration tool for IPv4/IPv6 packet filtering and NAT.
- [ ] **tcpdump**: Dump traffic on a network (would require stream capturing limitations).
