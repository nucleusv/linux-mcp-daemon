# Containers & Kubernetes Troubleshooting

This daemon's own real deployment (this entire investigation's environment) is a Kubernetes-in-Docker node — this category is included because it's the daemon's actual operating context, not generic "every Linux box" territory. See `plan/linux-admin-roadmap.md` item 8 for the scope caveat: this is a deliberate, separate decision (does this daemon's mandate extend to k8s-node-specific tooling?), not assumed by default.

## 1. "A pod is CrashLoopBackOff, how do you debug it?" (classic platform-engineering interview question)

**Ticket**: A Kubernetes pod keeps restarting.

**Real test**: This investigation's own real host runs `containerd` (PID 151) directly, with `containerd-shim-runc-v2` processes visible per-container in `processes/list` (e.g. PIDs 1156, 1297, 1323...). At the **host-process level**, `processes/list` can see these shims exist and are alive; `disks/mounts` can see each container's `overlayfs` root (`disk-storage.md` scenario 2's `io.containerd.snapshotter.v1.overlayfs`). But there's no way to map a shim PID back to a Kubernetes pod/container name, no way to read a specific container's exit code or last logs (as opposed to the *host's* systemd/journal logs), and no way to see `kubectl describe pod`-style events (`Back-off restarting failed container`, `ImagePullBackOff`, etc.).

**Verdict: Not solvable at the container/pod level.** The host-level tools correctly show that *something* is running, but the actual "why is this specific pod crash-looping" question needs container-runtime-aware tooling this daemon doesn't have.

## 2. List running containers (the `crictl ps`/`docker ps` equivalent)

**Ticket**: "What containers are actually running on this node?"

**Real test**: No tool exists. `processes/list` shows the host-process view (shims, not containers) - real containerd-namespace container IDs are visible only inside each shim's `cmdline` (e.g. `-id 26dbab36ed28066ea4464a4ae3a337a3149f92f6a121600e60bf449f4e849321`), not as a first-class field.

**Verdict: Not solvable**, matches roadmap item 8's proposed `containers/list` (talking to containerd's own gRPC socket, per that item's kernel-first-equivalent framing).

## 3. Check container resource usage (CPU/memory) per-container, not per-host

**Ticket**: A specific container is suspected of resource exhaustion, not the host as a whole.

**Real test**: `memory/usage` and the (broken) CPU-sort in `processes/list` (see `cpu-performance.md` scenario 2) are host-wide or host-process-wide, not cgroup-scoped. No tool reads `/sys/fs/cgroup/*/memory.current` or `cpu.stat` per-container.

**Verdict: Not solvable.** Same underlying gap as `cpu-performance.md` scenario 5 (no cgroup awareness anywhere in the toolset), specifically relevant here because every real workload on this host runs in a cgroup-scoped container.

## 4. Real, live example this investigation actually found: correlating a host port to a Kubernetes-managed process

**Ticket**: Confirm which Kubernetes control-plane component owns a given port.

**Real test**: `network/connections` (see `network.md` scenario 1) directly answered this for real, live control-plane ports on this host - e.g. `*:6443` → `kube-apiserver`, `*:10250` → `kubelet` - without any Kubernetes-specific tooling at all, just host-level socket-to-PID correlation plus recognizing the process names.

**Verdict: Solvable**, and a genuine positive example: for control-plane processes that run as plain host processes (true for `kubeadm`-style clusters like this one, not necessarily for managed/cloud Kubernetes), existing host-level tools already answer a meaningfully Kubernetes-relevant question without any dedicated k8s awareness.
