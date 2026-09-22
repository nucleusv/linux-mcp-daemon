package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/auth"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/config"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/devices/dmi"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/devices/pci"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/devices/usb"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/kernel/modules"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/resources/network/routes"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/rpc"
	cpulist "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/cpu/list"
	loadaverage "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/cpu/load-average"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/free"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/health"
	disklist "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/list"
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
	mem_usage "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/memory/usage"
	system_control "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/kernel/system-control"
	dmesg "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/logs/dmesg"
	journal_control "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/logs/journal-control"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/services/list"
	manage_service "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/services/manage"
	status_service "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/services/status"
	process_read "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/processes/read"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/arp"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/connections"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/curl"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/nslookup"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/ping"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/trace-path"
	deleteprocess "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/processes/delete"
	listprocesses "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/processes/list"
	osrelease "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/system/os-release"
)

func main() {
	// ==========================================
	// WORKER MODE (Ephemeral Execution)
	// ==========================================
	if len(os.Args) > 1 && os.Args[1] == "worker" {
		if len(os.Args) < 4 {
			log.Fatalf("Usage: mcpd worker <tool_name> <json_args>")
		}
		toolName := os.Args[2]
		toolArgs := []byte(os.Args[3])

		var result string
		var err error

		handlers := map[string]func([]byte) (string, error){
			"files/list":          listfiles.ListOfFiles,
			"disks/free":          free.Free,
			"disks/usage":         disk_usage.Usage,
			"processes/list":      listprocesses.Processes,
			"processes/delete":    deleteprocess.Delete,
			"network/connections": connections.Connections,
			"network/nslookup":    nslookup.Nslookup,
			"network/curl":        curl.Curl,
			"network/arp":         arp.ARP,
			"network/ping":        ping.Ping,
			"network/trace-path":  tracepath.TracePath,
			"memory/usage":        mem_usage.Usage,
			"cpu/list":            cpulist.List,
			"cpu/load-average":    loadaverage.LoadAverage,
			"disks/list":          disklist.List,
			"disks/performance":   performance.Performance,
			"disks/health":        health.Health,
			"disks/partitions":    partitions.Partitions,
			"system/os-release":   osrelease.OSRelease,
			"files/stat":          stat.Stat,
			"files/content":       content.Content,
			"files/read":          readfile.Read,
			"files/create":        createfile.Create,
			"files/update":        updatefile.Update,
			"files/find":          findfile.Find,
			"files/filetype":      filetype.Type,
			"read_usb":            usb.ReadUSB,
			"read_pci":            pci.ReadPCI,
			"read_dmi":            dmi.ReadDMI,
			"read_modules":        modules.ReadModules,
			"read_routes":         routes.ReadRoutes,
			"services/list":       list.List,
			"services/manage":     manage_service.Manage,
			"services/status":     status_service.Status,
			"logs/journal-control": journal_control.JournalControl,
			"logs/dmesg":          dmesg.Dmesg,
			"kernel/system-control": system_control.SystemControl,
			"processes/read":      process_read.Read,
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
	if err := loadConfig("configs/daemon.yaml"); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var err error
	sudoConfig, err = config.LoadSudoConfig("configs/mcp-sudo.yaml")
	if err != nil {
		log.Fatalf("Failed to load mcp-sudo.yaml: %v", err)
	}

	limiterManager = auth.NewLimiterManager(daemonConfig.RateLimits.DefaultRPS, daemonConfig.RateLimits.DefaultBurst)

	addr := fmt.Sprintf(":%d", daemonConfig.Server.Port)
	if daemonConfig.Server.Port == 0 {
		addr = ":9090"
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

	http.HandleFunc("/sse", handleSSE)
	http.HandleFunc("/message", handleMessage)

	// Serve documentation
	fs := http.FileServer(http.Dir("docs/website/build"))
	http.Handle("/docs/", http.StripPrefix("/docs/", fs))

	if daemonConfig.Server.TLS.Enabled {
		tlsAddr := fmt.Sprintf(":%d", daemonConfig.Server.TLS.Port)
		if daemonConfig.Server.TLS.Port == 0 {
			tlsAddr = ":9443"
		}

		go func() {
			log.Printf("Starting Linux MCP Daemon (HTTPS SSE Transport) on %s\n", tlsAddr)
			if err := http.ListenAndServeTLS(tlsAddr, daemonConfig.Server.TLS.CertFile, daemonConfig.Server.TLS.KeyFile, nil); err != nil {
				log.Fatalf("Daemon TLS crashed: %v", err)
			}
		}()
	}

	log.Printf("Starting Linux MCP Daemon (HTTP SSE Transport) on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Daemon crashed: %v", err)
	}
}
