package main

import (
	"crypto/tls"
	"github.com/nucleusv/linux-mcp-daemon/internal/version"
	"path/filepath"

	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/nucleusv/linux-mcp-daemon/internal/auth"
	"github.com/nucleusv/linux-mcp-daemon/internal/config"
	"github.com/nucleusv/linux-mcp-daemon/internal/logging"
	"github.com/nucleusv/linux-mcp-daemon/internal/resources/devices/dmi"
	"github.com/nucleusv/linux-mcp-daemon/internal/resources/devices/pci"
	"github.com/nucleusv/linux-mcp-daemon/internal/resources/devices/usb"
	"github.com/nucleusv/linux-mcp-daemon/internal/resources/kernel/modules"
	"github.com/nucleusv/linux-mcp-daemon/internal/resources/network/interfaces"
	"github.com/nucleusv/linux-mcp-daemon/internal/resources/network/routes"
	"github.com/nucleusv/linux-mcp-daemon/internal/rpc"
	"github.com/nucleusv/linux-mcp-daemon/internal/tlsutil"
	cpulist "github.com/nucleusv/linux-mcp-daemon/internal/tools/cpu/list"
	loadaverage "github.com/nucleusv/linux-mcp-daemon/internal/tools/cpu/load-average"
	"github.com/nucleusv/linux-mcp-daemon/internal/tools/disks/free"
	"github.com/nucleusv/linux-mcp-daemon/internal/tools/disks/health"
	disklist "github.com/nucleusv/linux-mcp-daemon/internal/tools/disks/list"
	diskmounts "github.com/nucleusv/linux-mcp-daemon/internal/tools/disks/mounts"
	"github.com/nucleusv/linux-mcp-daemon/internal/tools/disks/partitions"
	"github.com/nucleusv/linux-mcp-daemon/internal/tools/disks/performance"
	disk_usage "github.com/nucleusv/linux-mcp-daemon/internal/tools/disks/usage"
	chmodfile "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/chmod"
	chownfile "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/chown"
	content "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/content"
	createfile "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/create"
	filetype "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/filetype"
	findfile "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/find"
	listfiles "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/list"
	readfile "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/read"
	stat "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/stat"
	updatefile "github.com/nucleusv/linux-mcp-daemon/internal/tools/files/update"
	system_control "github.com/nucleusv/linux-mcp-daemon/internal/tools/kernel/system-control"
	dmesg "github.com/nucleusv/linux-mcp-daemon/internal/tools/logs/dmesg"
	journal_control "github.com/nucleusv/linux-mcp-daemon/internal/tools/logs/journal-control"
	logins "github.com/nucleusv/linux-mcp-daemon/internal/tools/logs/logins"
	mem_usage "github.com/nucleusv/linux-mcp-daemon/internal/tools/memory/usage"
	"github.com/nucleusv/linux-mcp-daemon/internal/tools/network/arp"
	"github.com/nucleusv/linux-mcp-daemon/internal/tools/network/connections"
	"github.com/nucleusv/linux-mcp-daemon/internal/tools/network/curl"
	"github.com/nucleusv/linux-mcp-daemon/internal/tools/network/nslookup"
	"github.com/nucleusv/linux-mcp-daemon/internal/tools/network/ping"
	"github.com/nucleusv/linux-mcp-daemon/internal/tools/network/trace-path"
	deleteprocess "github.com/nucleusv/linux-mcp-daemon/internal/tools/processes/delete"
	listprocesses "github.com/nucleusv/linux-mcp-daemon/internal/tools/processes/list"
	process_read "github.com/nucleusv/linux-mcp-daemon/internal/tools/processes/read"
	topprocesses "github.com/nucleusv/linux-mcp-daemon/internal/tools/processes/top"
	"github.com/nucleusv/linux-mcp-daemon/internal/tools/services/list"
	manage_service "github.com/nucleusv/linux-mcp-daemon/internal/tools/services/manage"
	status_service "github.com/nucleusv/linux-mcp-daemon/internal/tools/services/status"
	osrelease "github.com/nucleusv/linux-mcp-daemon/internal/tools/system/os-release"
	"github.com/nucleusv/linux-mcp-daemon/internal/tools/system/packages"
	userslist "github.com/nucleusv/linux-mcp-daemon/internal/tools/users/list"
	"github.com/nucleusv/linux-mcp-daemon/internal/worker"
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
				fmt.Fprintf(os.Stderr, "cannot switch to the host's filesystem - is the mcpd container running with --privileged --pid host? (%v)\n", err)
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
			"files/chmod":           chmodfile.Chmod,
			"files/chown":           chownfile.Chown,
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
	if dir := os.Getenv("MCPD_CONFIG_DIR"); dir != "" {
		configDir = dir
	}
	for i := 1; i < len(os.Args); i++ {
		switch {
		case os.Args[i] == "--config-dir" && i+1 < len(os.Args):
			configDir = os.Args[i+1]
			i++
		case strings.HasPrefix(os.Args[i], "--config-dir="):
			configDir = strings.TrimPrefix(os.Args[i], "--config-dir=")
		default:
			logging.Fatal("unknown argument (usage: mcpd [--config-dir DIR] | mcpd --version)", "arg", os.Args[i])
		}
	}

	var err error
	daemonConfig, usersPath, err = loadConfig()
	if err != nil {
		logging.Fatal("cannot load config", "err", err)
	}
	logging.Configure(daemonConfig.Logging)
	logging.Info("config loaded", "dir", configDir, "log_level", logLevel(daemonConfig))
	if w := legacyUsersWarning(daemonConfig, usersPath); w != "" {
		logging.Warn(w)
	}

	// As for daemon.yaml: strict problems (unknown keys, paths: on a tool
	// that takes no path) are warnings at startup, errors on reload.
	sudoConfig, err := config.LoadSudoConfigStrict(sudoConfigPath())
	if err != nil {
		strictErr := err
		if sudoConfig, err = config.LoadSudoConfig(sudoConfigPath()); err != nil {
			logging.Fatal("cannot load mcp-sudo.yaml", "err", err)
		}
		logging.Warn("mcp-sudo.yaml loaded leniently; daemon/reload-config and linuxctl edit will refuse it until it's fixed", "err", strictErr)
	}

	worker.Containerized = daemonConfig.Worker.Containerized
	if actual, err := worker.IsContainerized(); err != nil {
		logging.Warn("could not determine whether this process is actually containerized (comparing /proc/self/ns/mnt vs /proc/1/ns/mnt)", "err", err)
	} else if actual != daemonConfig.Worker.Containerized {
		if daemonConfig.Worker.Containerized {
			logging.Warn("daemon.yaml sets worker.containerized: true, but this process shares its mount namespace with its own PID 1 - no host is in view, so privileged calls will fail. In a container, start it with --privileged --pid host; if mcpd runs directly on the host, set worker.containerized: false.")
		} else {
			logging.Warn("daemon.yaml sets worker.containerized: false, but this process appears to be running in its own mount namespace, separate from its own PID 1 - privileged tools like system/packages or services/manage will only see this container's own filesystem, not the real host's. If mcpd is deployed containerized (e.g. Kubernetes, Docker) and should administer the real host, set worker.containerized: true.")
		}
	}

	limiterManager.Store(auth.NewLimiterManager(daemonConfig.RateLimits.DefaultRPS, daemonConfig.RateLimits.DefaultBurst))

	rpcHandler = rpc.NewRPCHandler(
		sudoConfig,
		daemonConfig.Worker.TimeoutSeconds,
		daemonConfig.Tools,
		&requestGroup,
		rpcCache,
		&cacheMu,
		resourceCache,
	)
	rpcHandler.ReloadConfig = reloadConfig

	mux := http.NewServeMux()
	mux.HandleFunc("/sse", handleSSE)
	mux.HandleFunc("/message", handleMessage)

	handler := loggingMiddleware(mux)

	serve(handler)
}

// serve starts every configured listener - TLS (with a generated
// self-signed certificate when configured and missing) and/or plain HTTP -
// and blocks; any listener failing stops mcpd.
func serve(handler http.Handler) {
	errs := make(chan error, 2)
	for _, l := range daemonConfig.Listeners() {
		addr := fmt.Sprintf(":%d", l.Port)
		if !l.TLS {
			logging.Warn("serving plain HTTP: bearer tokens travel in clear text - use TLS, or keep this port to a trusted network", "addr", addr)
			logging.Info("listening", "version", version.Version, "transport", "http", "addr", addr)
			go func() { errs <- fmt.Errorf("HTTP on %s: %w", addr, http.ListenAndServe(addr, handler)) }()
			continue
		}
		certFile, keyFile := daemonConfig.TLSFiles(configDir)
		if abs, err := filepath.Abs(certFile); err == nil {
			certFile = abs
		}
		if abs, err := filepath.Abs(keyFile); err == nil {
			keyFile = abs
		}
		if daemonConfig.Server.TLS.Generate {
			created, err := tlsutil.EnsureSelfSigned(certFile, keyFile, daemonConfig.Server.TLS.Hosts)
			if err != nil {
				logging.Fatal("cannot create the TLS certificate", "cert", certFile, "err", err)
			}
			if created {
				logging.Info("created a self-signed TLS certificate", "cert", certFile, "key", keyFile)
			}
		}
		fp, cert, err := tlsutil.FileFingerprint(certFile)
		if err != nil {
			logging.Fatal("cannot read the TLS certificate", "cert", certFile, "err", err)
		}
		logging.Info("listening", "version", version.Version, "transport", "https", "addr", addr,
			"cert", certFile, "fingerprint", fp, "expires", cert.NotAfter.Format("2006-01-02"))
		srv := &http.Server{Addr: addr, Handler: handler, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
		go func() { errs <- fmt.Errorf("HTTPS on %s: %w", addr, srv.ListenAndServeTLS(certFile, keyFile)) }()
	}
	err := <-errs
	logging.Fatal("cannot serve", "err", err)
}
