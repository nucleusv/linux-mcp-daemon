# Processes Group Implementation Plan

This document tracks the standard Linux process utilities and their implementation status in the MCP Daemon.

## Process Management

- [x] **ps** (`processes/list-processes`): Report a snapshot of the current processes.
- [x] **kill** (`processes/kill-process`): Send a signal to a process.
- [ ] **top / htop**: Display Linux processes dynamically (might be tricky over stateless RPC, but a one-shot snapshot could be implemented).
- [ ] **pgrep / pkill**: Look up or signal processes based on name and other attributes.
- [ ] **killall**: Kill processes by name.
- [ ] **fuser**: Identify processes using files or sockets.
- [ ] **renice**: Alter priority of running processes.
