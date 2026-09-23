package main

import (
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/version"

	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/auth"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/devices/dmi"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/devices/pci"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/devices/usb"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/kernel/modules"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/network/interfaces"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/network/routes"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/rpc"
	cpulist "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/cpu/list"
	loadaverage "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/cpu/load-average"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/free"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/health"
	disklist "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/list"
	diskmounts "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/mounts"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/partitions"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/performance"
	disk_usage "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/usage"
	content "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/files/content"
	createfile "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/files/create"
	filetype "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/files/filetype"
	findfile "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/files/find"
	listfiles "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/files/list"
	readfile "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/files/read"
	stat "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/files/stat"
	updatefile "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/files/update"
	system_control "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/kernel/system-control"
	dmesg "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/logs/dmesg"
	journal_control "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/logs/journal-control"
	logins "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/logs/logins"
	mem_usage "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/memory/usage"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/arp"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/connections"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/curl"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/nslookup"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/ping"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/trace-path"
	deleteprocess "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/processes/delete"
	listprocesses "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/processes/list"
	process_read "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/processes/read"
	topprocesses "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/processes/top"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/services/list"
	manage_service "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/services/manage"
	status_service "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/services/status"
	osrelease "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/system/os-release"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/system/packages"
	userslist "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/users/list"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/worker"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-version" || os.Args[1] == "version") {
		fmt.Println(version.String("mcpd"))
		return
	}

	// ==========================================
	// WORKER MODE (Ephemeral Execution)
	// ==========================================
	if len(os.Args) > 1 && os.Args[1] == "worker" {
		if len(os.Args) < 3 {
			log.Fatalf("Usage: mcpd worker <tool_name> [json_args]   (json_args read from stdin when omitted)")
		}
		toolName := os.Args[2]
		// The daemon sends arguments on stdin (argv is world-readable via
		// /proc/<pid>/cmdline); a 4th argv is still accepted for manual
		// debugging.
		var toolArgs []byte
		if len(os.Args) >= 4 {
			toolArgs = []byte(os.Args[3])
		} else {
			b, err := io.ReadAll(io.LimitReader(os.Stdin, 64<<20))
			if err != nil {
				log.Fatalf("failed to read worker arguments from stdin: %v", err)
			}
			toolArgs = b
		}

		if os.Getenv("MCPD_HOST_ROOT") == "1" {
			if err := worker.JoinHostMountNamespace(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to join host mount namespace: %v\n", err)
				os.Exit(1)
			}
		}

		var result string
		var err error

		handlers := map[string]func([]byte) (string, error){
			"files/list":            listfiles.ListOfFiles,
			"disks/free":            free.Free,
			"disks/usage":           disk_usage.Usage,
			"processes/list":        listprocesses.Processes,
			"processes/top":         topprocesses.Top,
			"processes/delete":      deleteprocess.Delete,
			"network/connections":   connections.Connections,
			"network/nslookup":      nslookup.Nslookup,
			"network/curl":          curl.Curl,
			"network/arp":           arp.ARP,
			"network/ping":          ping.Ping,
			"network/trace-path":    tracepath.TracePath,
			"memory/usage":          mem_usage.Usage,
			"cpu/list":              cpulist.List,
			"cpu/load-average":      loadaverage.LoadAverage,
			"disks/list":            disklist.List,
			"disks/mounts":          diskmounts.List,
			"disks/performance":     performance.Performance,
			"disks/health":          health.Health,
			"disks/partitions":      partitions.Partitions,
			"system/os-release":     osrelease.OSRelease,
			"system/packages":       packages.List,
			"users/list":            userslist.List,
			"files/stat":            stat.Stat,
			"files/content":         content.Content,
			"files/read":            readfile.Read,
			"files/create":          createfile.Create,
			"files/update":          updatefile.Update,
			"files/find":            findfile.Find,
			"files/filetype":        filetype.Type,
			"read_usb":              usb.ReadUSB,
			"read_pci":              pci.ReadPCI,
			"read_dmi":              dmi.ReadDMI,
			"read_modules":          modules.ReadModules,
			"read_routes":           routes.ReadRoutes,
			"read_interfaces":       interfaces.ReadWorker,
			"services/list":         list.List,
			"services/manage":       manage_service.Manage,
			"services/status":       status_service.Status,
			"logs/journal-control":  journal_control.JournalControl,
			"logs/logins":           logins.List,
			"logs/dmesg":            dmesg.Dmesg,
			"kernel/system-control": system_control.SystemControl,
			"processes/read":        process_read.Read,
		}

		if handler, exists := handlers[toolName]; exists {
			result, err = handler(toolArgs)
		} else {
			log.Fatalf("Unknown tool: %s", toolName)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Print(result)
		os.Exit(0)
	}

	// ==========================================
	// MASTER DAEMON MODE
	// ==========================================
	daemonConfigPath = "configs/daemon.yaml"
	if err := loadConfig(daemonConfigPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var err error
	sudoConfig, err = config.LoadSudoConfig("configs/mcp-sudo.yaml")
	if err != nil {
		log.Fatalf("Failed to load mcp-sudo.yaml: %v", err)
	}

	worker.Containerized = daemonConfig.Worker.Containerized
	if actual, err := worker.IsContainerized(); err != nil {
		log.Printf("WARNING: could not determine whether this process is actually containerized (comparing /proc/self/ns/mnt vs /proc/1/ns/mnt): %v", err)
	} else if actual != daemonConfig.Worker.Containerized {
		if daemonConfig.Worker.Containerized {
			log.Printf("WARNING: configs/daemon.yaml sets worker.containerized: true, but this process does not appear to be in a separate mount namespace from its own PID 1 - there may be no real container boundary to cross. This is harmless on its own (JoinHostMountNamespace no-ops when the namespace already matches), but if mcpd actually runs directly on the host, set worker.containerized: false to skip the redundant check on every privileged call.")
		} else {
			log.Printf("WARNING: configs/daemon.yaml sets worker.containerized: false, but this process appears to be running in its own mount namespace, separate from its own PID 1 - privileged tools like system/packages or services/manage will only see this container's own filesystem, not the real host's. If mcpd is deployed containerized (e.g. Kubernetes, Docker) and should administer the real host, set worker.containerized: true.")
		}
	}

	limiterManager = auth.NewLimiterManager(daemonConfig.RateLimits.DefaultRPS, daemonConfig.RateLimits.DefaultBurst)

	addr := fmt.Sprintf(":%d", daemonConfig.Server.Port)
	if daemonConfig.Server.Port == 0 {
		addr = ":9091"
	}
	if daemonConfig.Worker.TimeoutSeconds == 0 {
		daemonConfig.Worker.TimeoutSeconds = 30
	}

	rpcHandler = rpc.NewRPCHandler(
		sudoConfig,
		daemonConfig.Worker.TimeoutSeconds,
		daemonConfig.Tools,
		&requestGroup,
		rpcCache,
		&cacheMu,
		resourceCache,
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/sse", handleSSE)
	mux.HandleFunc("/message", handleMessage)

	// Serve documentation
	fs := http.FileServer(http.Dir("docs/website/build"))
	mux.Handle("/docs/", http.StripPrefix("/docs/", fs))

	handler := loggingMiddleware(mux)

	if daemonConfig.Server.TLS.Enabled {
		tlsAddr := fmt.Sprintf(":%d", daemonConfig.Server.TLS.Port)
		if daemonConfig.Server.TLS.Port == 0 {
			tlsAddr = ":9443"
		}

		go func() {
			log.Printf("Starting Linux MCP Daemon %s (HTTPS SSE Transport) on %s\n", version.Version, tlsAddr)
			if err := http.ListenAndServeTLS(tlsAddr, daemonConfig.Server.TLS.CertFile, daemonConfig.Server.TLS.KeyFile, handler); err != nil {
				log.Fatalf("Daemon TLS crashed: %v", err)
			}
		}()
	}

	log.Printf("Starting Linux MCP Daemon %s (HTTP SSE Transport) on %s\n", version.Version, addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Daemon crashed: %v", err)
	}
}
