package listprocesses

import (
	"encoding/json"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/procstat"

	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"
)

type GetProcessesArgs struct {
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format (e.g. "json"). Defaults to text.
	User         string `json:"user"`
	PID          int    `json:"pid"`
	SortBy       string `json:"sort_by"`
	Limit        int    `json:"limit"`
	Privileged   bool   `json:"privileged"`
}

type Process struct {
	PID     int    `json:"pid"`
	User    string `json:"user"`
	Comm    string `json:"comm"`
	State   string `json:"state"`
	PPID    int    `json:"ppid"`
	RSS     int64  `json:"rss_kb"`
	Cmdline string `json:"cmdline"`
	// CPUPercent is only measured for sort_by: cpu (it needs two samples
	// over an interval, which costs half a second).
	CPUPercent *float64 `json:"cpu_percent,omitempty"`
}

// cpuSampleInterval is how long sort_by: cpu measures over.
const cpuSampleInterval = 500 * time.Millisecond

func Processes(argsJSON []byte) (string, error) {
	var args GetProcessesArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	// sort_by: cpu was advertised in the schema but never implemented.
	// %CPU needs two samples: CPU ticks now, and again after an interval.
	var cpuBefore map[int]uint64
	var sampleStart time.Time
	if args.SortBy == "cpu" {
		cpuBefore = map[int]uint64{}
		if pids, err := procstat.ListPIDs(); err == nil {
			for _, pid := range pids {
				if p, err := procstat.ReadProc(pid); err == nil {
					cpuBefore[pid] = p.CPUTicks()
				}
			}
		}
		sampleStart = time.Now()
		time.Sleep(cpuSampleInterval)
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc: %v", err)
	}

	var procs []Process
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}

		p := Process{PID: pid}

		// Read stat
		statData, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err == nil {
			statStr := string(statData)
			commStart := strings.IndexByte(statStr, '(')
			commEnd := strings.LastIndexByte(statStr, ')')
			if commStart != -1 && commEnd != -1 && commEnd > commStart {
				p.Comm = statStr[commStart+1 : commEnd]
				fields := strings.Fields(statStr[commEnd+2:])
				if len(fields) >= 22 {
					p.State = fields[0]
					p.PPID, _ = strconv.Atoi(fields[1])
					rssPages, _ := strconv.ParseInt(fields[21], 10, 64)
					p.RSS = (rssPages * 4096) / 1024 // KB
				}
			}
		}

		// Read cmdline
		cmdData, _ := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		cmdDataStr := strings.ReplaceAll(string(cmdData), "\x00", " ")
		p.Cmdline = strings.TrimSpace(cmdDataStr)

		// Read status for UID
		statusData, _ := ioutil.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
		for _, line := range strings.Split(string(statusData), "\n") {
			if strings.HasPrefix(line, "Uid:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					uidStr := fields[1]
					p.User = uidStr
					if u, err := user.LookupId(uidStr); err == nil {
						p.User = u.Username
					}
				}
				break
			}
		}

		// Filter by user if requested
		if args.User != "" && p.User != args.User && strconv.Itoa(p.PID) != args.User {
			continue
		}

		// Filter to a single specific PID if requested (linuxctl's `get
		// processes <pid>` - the "one result" form of the same underlying
		// list call, mirroring `kubectl get pod x` reusing `kubectl get
		// pods`'s row format rather than being a separate endpoint).
		if args.PID != 0 && p.PID != args.PID {
			continue
		}

		if cpuBefore != nil {
			if cur, err := procstat.ReadProc(pid); err == nil && cur.CPUTicks() >= cpuBefore[pid] {
				pct := float64(cur.CPUTicks()-cpuBefore[pid]) / procstat.ClockTicks / time.Since(sampleStart).Seconds() * 100
				pct = float64(int(pct*10+0.5)) / 10
				p.CPUPercent = &pct
			}
		}

		procs = append(procs, p)
	}

	// Sort - by PID unless asked otherwise, like ps. /proc's own directory
	// order is lexical ("1", "10014", "104", ...), never numeric.
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].PID < procs[j].PID
	})
	switch args.SortBy {
	case "mem":
		sort.SliceStable(procs, func(i, j int) bool {
			return procs[i].RSS > procs[j].RSS // Descending
		})
	case "cpu":
		sort.SliceStable(procs, func(i, j int) bool {
			return cpuOf(procs[i]) > cpuOf(procs[j]) // Descending
		})
	}

	// Limit
	if args.Limit > 0 && len(procs) > args.Limit {
		procs = procs[:args.Limit]
	}

	j, err := json.Marshal(procs)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	}

	return string(j), nil
}

func cpuOf(p Process) float64 {
	if p.CPUPercent == nil {
		return 0
	}
	return *p.CPUPercent
}
