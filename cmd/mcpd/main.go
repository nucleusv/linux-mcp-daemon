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

		if toolName == "files/list" {
			result, err = listfiles.ListOfFiles(toolArgs)
		} else if toolName == "disks/free" {
			result, err = free.Free(toolArgs)
		} else if toolName == "disks/usage" {
			result, err = disk_usage.Usage(toolArgs)
		} else if toolName == "processes/list" {
			result, err = listprocesses.Processes(toolArgs)
		} else if toolName == "processes/delete" {
			result, err = deleteprocess.Delete(toolArgs)
		} else if toolName == "network/connections" {
			result, err = connections.Connections(toolArgs)
		} else if toolName == "network/nslookup" {
			result, err = nslookup.Nslookup(toolArgs)
		} else if toolName == "network/curl" {
			result, err = curl.Curl(toolArgs)
		} else if toolName == "network/arp" {
			result, err = arp.ARP(toolArgs)
		} else if toolName == "network/ping" {
			result, err = ping.Ping(toolArgs)
		} else if toolName == "memory/usage" {
			result, err = mem_usage.Usage(toolArgs)
		} else if toolName == "cpu/info" {
			result, err = info.Info(toolArgs)
		} else if toolName == "cpu/load-average" {
			result, err = loadaverage.LoadAverage(toolArgs)
		} else if toolName == "disks/block-devices" {
			result, err = blockdevices.GetBlockDevices(toolArgs)
		} else if toolName == "system/os-release" {
			result, err = osrelease.OSRelease(toolArgs)
		} else if toolName == "files/read" {
			result, err = readfile.Read(toolArgs)

		} else if toolName == "read_usb" {
			result, err = usb.ReadUSB(toolArgs)
		} else if toolName == "read_pci" {
			result, err = pci.ReadPCI(toolArgs)
		} else if toolName == "read_dmi" {
			result, err = dmi.ReadDMI(toolArgs)
		} else if toolName == "read_modules" {
			result, err = modules.ReadModules(toolArgs)
		} else if toolName == "read_routes" {
			result, err = routes.ReadRoutes(toolArgs)
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
