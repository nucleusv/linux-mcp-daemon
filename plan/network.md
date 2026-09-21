# Network Group Implementation Plan

This document tracks the standard Linux network utilities and their implementation status in the MCP Daemon.

## Network Interfaces and Sockets

- [x] **ip address** (`network/list-interfaces`): Show / manipulate routing, network devices, interfaces and tunnels.
- [x] **ss / netstat** (`network/list-connections`): Utility to investigate sockets.
- [ ] **ping**: Send ICMP ECHO_REQUEST to network hosts.
- [ ] **traceroute / tracepath**: Print the route packets trace to network host.
- [ ] **nslookup / dig**: Query Internet name servers interactively.
- [ ] **curl / wget**: Transfer a URL (could be useful for MCP to verify internal endpoints).
- [ ] **arp / ip neigh**: Manipulate or view the system ARP cache.
- [ ] **iptables / nft**: Administration tool for IPv4/IPv6 packet filtering and NAT.
- [ ] **tcpdump**: Dump traffic on a network (would require stream capturing limitations).
