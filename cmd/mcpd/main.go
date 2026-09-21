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
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/cpu/info"
	loadaverage "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/cpu/load-average"
	blockdevices "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/block-devices"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/free"
	disk_usage "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/disks/usage"
	listfiles "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/files/list"
	readfile "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/files/read"
	mem_usage "github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/memory/usage"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/arp"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/connections"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/curl"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/nslookup"
	"github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/tools/network/ping"
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
			"memory/usage":        mem_usage.Usage,
			"cpu/info":            info.Info,
			"cpu/load-average":    loadaverage.LoadAverage,
			"disks/block-devices": blockdevices.GetBlockDevices,
			"system/os-release":   osrelease.OSRelease,
			"files/read":          readfile.Read,
			"read_usb":            usb.ReadUSB,
			"read_pci":            pci.ReadPCI,
			"read_dmi":            dmi.ReadDMI,
			"read_modules":        modules.ReadModules,
			"read_routes":         routes.ReadRoutes,
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
