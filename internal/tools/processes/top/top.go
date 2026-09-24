// Package top implements processes/top: a one-shot equivalent of the top
// command's batch output (`top -b -n 1`) - the summary header followed by
// the process table with all of top's default columns - read natively
// from /proc.
package top

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/procstat"
)

// TopArgs defines the parameters for the processes/top tool.
type TopArgs struct {
	IntervalMs   int    `json:"interval_ms,omitempty"`   // Sampling interval for %CPU. Default 1000, max 10000.
	SortBy       string `json:"sort_by,omitempty"`       // cpu (default), mem, res, time, pid.
	Limit        int    `json:"limit,omitempty"`         // Max processes to list; 0 = all.
	User         string `json:"user,omitempty"`          // Only processes of this user.
	OutputFormat string `json:"output_format,omitempty"` // json/yaml/table/wide for structured output.
	Privileged   bool   `json:"privileged,omitempty"`
}

// Summary is top's header.
type Summary struct {
	Time        string     `json:"time"`
	Uptime      string     `json:"uptime"`
	UptimeSecs  int64      `json:"uptime_seconds"`
	Users       int        `json:"users"`
	LoadAverage [3]float64 `json:"load_average"`
	Tasks       Tasks      `json:"tasks"`
	CPU         CPUPercent `json:"cpu_percent"`
	MemMiB      MemMiB     `json:"mem_mib"`
	SwapMiB     SwapMiB    `json:"swap_mib"`
}

type Tasks struct {
	Total    int `json:"total"`
	Running  int `json:"running"`
	Sleeping int `json:"sleeping"`
	Stopped  int `json:"stopped"`
	Zombie   int `json:"zombie"`
}

type CPUPercent struct {
	User    float64 `json:"us"`
	System  float64 `json:"sy"`
	Nice    float64 `json:"ni"`
	Idle    float64 `json:"id"`
	IOWait  float64 `json:"wa"`
	IRQ     float64 `json:"hi"`
	SoftIRQ float64 `json:"si"`
	Steal   float64 `json:"st"`
}

type MemMiB struct {
	Total     float64 `json:"total"`
	Free      float64 `json:"free"`
	Used      float64 `json:"used"`
	BuffCache float64 `json:"buff_cache"`
}

type SwapMiB struct {
	Total    float64 `json:"total"`
	Free     float64 `json:"free"`
	Used     float64 `json:"used"`
	AvailMem float64 `json:"avail_mem"`
}

// Row is one process line, with top's default columns.
type Row struct {
	PID        int     `json:"pid"`
	User       string  `json:"user"`
	PR         string  `json:"pr"`
	NI         int64   `json:"ni"`
	VirtKiB    uint64  `json:"virt_kib"`
	ResKiB     uint64  `json:"res_kib"`
	ShrKiB     uint64  `json:"shr_kib"`
	State      string  `json:"s"`
	CPUPercent float64 `json:"cpu_percent"`
	MemPercent float64 `json:"mem_percent"`
	TimePlus   string  `json:"time_plus"`
	Command    string  `json:"command"`
	Cmdline    string  `json:"cmdline,omitempty"`
	PPID       int     `json:"ppid"`
	Threads    int64   `json:"threads"`

	cpuTicks uint64
}

// Snapshot is the full result.
type Snapshot struct {
	Summary   Summary `json:"summary"`
	Processes []Row   `json:"processes"`
}

func Top(argsJSON []byte) (string, error) {
	var args TopArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}
	interval := args.IntervalMs
	if interval <= 0 {
		interval = 1000
	}
	if interval > 10000 {
		interval = 10000
	}
	switch args.SortBy {
	case "", "cpu", "mem", "res", "time", "pid":
	default:
		return "", fmt.Errorf("invalid sort_by %q (cpu, mem, res, time, pid)", args.SortBy)
	}

	snap, err := Take(time.Duration(interval) * time.Millisecond)
	if err != nil {
		return "", err
	}
	if args.User != "" {
		var kept []Row
		for _, r := range snap.Processes {
			if r.User == args.User {
				kept = append(kept, r)
			}
		}
		snap.Processes = kept
	}
	SortRows(snap.Processes, args.SortBy)
	if args.Limit > 0 && len(snap.Processes) > args.Limit {
		snap.Processes = snap.Processes[:args.Limit]
	}
	if snap.Processes == nil {
		snap.Processes = []Row{}
	}

	switch args.OutputFormat {
	case "json", "yaml":
		b, err := json.Marshal(snap)
		if err != nil {
			return "", err
		}
		return string(b), nil
	case "wide":
		// top's own layout already is the table; wide adds PPID/THR and
		// the full command line, like `top -c`.
		return FormatWide(snap), nil
	}
	return Format(snap), nil
}

// Take samples the system twice, interval apart, so %CPU reflects recent
// activity rather than the average since boot - the way top computes it.
func Take(interval time.Duration) (Snapshot, error) {
	cpu1, err := procstat.ReadCPU()
	if err != nil {
		return Snapshot{}, fmt.Errorf("reading /proc/stat: %v", err)
	}
	pids, err := procstat.ListPIDs()
	if err != nil {
		return Snapshot{}, fmt.Errorf("reading /proc: %v", err)
	}
	before := make(map[int]uint64, len(pids))
	for _, pid := range pids {
		if p, err := procstat.ReadProc(pid); err == nil {
			before[pid] = p.CPUTicks()
		}
	}
	start := time.Now()
	time.Sleep(interval)
	elapsed := time.Since(start).Seconds()

	cpu2, err := procstat.ReadCPU()
	if err != nil {
		return Snapshot{}, err
	}
	mem, err := procstat.ReadMemory()
	if err != nil {
		return Snapshot{}, fmt.Errorf("reading /proc/meminfo: %v", err)
	}
	load, _ := procstat.ReadLoadAvg()
	uptime, _ := procstat.ReadUptime()
	names := procstat.Usernames()

	pids, err = procstat.ListPIDs()
	if err != nil {
		return Snapshot{}, err
	}
	var rows []Row
	var tasks Tasks
	for _, pid := range pids {
		p, err := procstat.ReadProc(pid)
		if err != nil {
			continue // exited between listing and reading
		}
		tasks.Total++
		switch p.State {
		case 'R':
			tasks.Running++
		case 'S', 'D', 'I':
			tasks.Sleeping++
		case 'T', 't':
			tasks.Stopped++
		case 'Z':
			tasks.Zombie++
		}

		// A process that started during the interval has no "before"
		// sample; its whole CPU time so far happened within the interval.
		delta := p.CPUTicks() - before[pid]
		if p.CPUTicks() < before[pid] {
			delta = 0 // PID reused by a new process
		}
		row := Row{
			PID:        p.PID,
			User:       userName(names, p.EUID),
			PR:         priority(p.Priority),
			NI:         p.Nice,
			VirtKiB:    p.VirtKiB,
			ResKiB:     p.ResKiB,
			ShrKiB:     p.ShrKiB,
			State:      string(p.State),
			CPUPercent: round1(float64(delta) / procstat.ClockTicks / elapsed * 100),
			TimePlus:   timePlus(p.CPUTicks()),
			Command:    p.Comm,
			Cmdline:    procstat.Cmdline(p.PID),
			PPID:       p.PPID,
			Threads:    p.Threads,
			cpuTicks:   p.CPUTicks(),
		}
		if mem.Total > 0 {
			row.MemPercent = round1(float64(p.ResKiB) / float64(mem.Total) * 100)
		}
		rows = append(rows, row)
	}

	return Snapshot{
		Summary: Summary{
			Time:        time.Now().Format("15:04:05"),
			Uptime:      formatUptime(int64(uptime)),
			UptimeSecs:  int64(uptime),
			Users:       procstat.CountUsers(),
			LoadAverage: load,
			Tasks:       tasks,
			CPU:         cpuPercent(cpu1, cpu2),
			MemMiB: MemMiB{
				Total:     mib(mem.Total),
				Free:      mib(mem.Free),
				Used:      mib(mem.Used()),
				BuffCache: mib(mem.BuffCache()),
			},
			SwapMiB: SwapMiB{
				Total:    mib(mem.SwapTotal),
				Free:     mib(mem.SwapFree),
				Used:     mib(mem.SwapTotal - mem.SwapFree),
				AvailMem: mib(mem.Available),
			},
		},
		Processes: rows,
	}, nil
}

// SortRows orders rows like top: by the chosen column descending (pid
// ascending), ties broken by PID.
func SortRows(rows []Row, by string) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		switch by {
		case "pid":
			return a.PID < b.PID
		case "mem", "res":
			if a.ResKiB != b.ResKiB {
				return a.ResKiB > b.ResKiB
			}
		case "time":
			if a.cpuTicks != b.cpuTicks {
				return a.cpuTicks > b.cpuTicks
			}
		default: // cpu
			if a.CPUPercent != b.CPUPercent {
				return a.CPUPercent > b.CPUPercent
			}
		}
		return a.PID < b.PID
	})
}

func cpuPercent(a, b procstat.CPUTimes) CPUPercent {
	total := float64(b.Total() - a.Total())
	if total <= 0 {
		return CPUPercent{Idle: 100}
	}
	pct := func(x, y uint64) float64 {
		if y < x {
			return 0
		}
		return round1(float64(y-x) / total * 100)
	}
	return CPUPercent{
		User:    pct(a.User, b.User),
		System:  pct(a.System, b.System),
		Nice:    pct(a.Nice, b.Nice),
		Idle:    pct(a.Idle, b.Idle),
		IOWait:  pct(a.IOWait, b.IOWait),
		IRQ:     pct(a.IRQ, b.IRQ),
		SoftIRQ: pct(a.SoftIRQ, b.SoftIRQ),
		Steal:   pct(a.Steal, b.Steal),
	}
}

func userName(names map[int]string, uid int) string {
	if n, ok := names[uid]; ok {
		return n
	}
	if uid < 0 {
		return "?"
	}
	return strconv.Itoa(uid)
}

// priority renders the PR column: real-time processes at the top priority
// show as "rt", as in top.
func priority(p int64) string {
	if p <= -100 {
		return "rt"
	}
	return strconv.FormatInt(p, 10)
}

// timePlus renders CPU time as top's TIME+ column: minutes:seconds.hundredths.
func timePlus(ticks uint64) string {
	cs := ticks * 100 / procstat.ClockTicks
	return fmt.Sprintf("%d:%02d.%02d", cs/6000, (cs/100)%60, cs%100)
}

// formatUptime renders uptime like top's header: "up 5 days,  3:02" or
// "up 42 min".
func formatUptime(secs int64) string {
	days := secs / 86400
	hours := (secs % 86400) / 3600
	mins := (secs % 3600) / 60
	s := "up "
	if days > 0 {
		unit := "days"
		if days == 1 {
			unit = "day"
		}
		s += fmt.Sprintf("%d %s, ", days, unit)
	}
	if hours > 0 {
		s += fmt.Sprintf("%2d:%02d", hours, mins)
	} else {
		s += fmt.Sprintf("%d min", mins)
	}
	return s
}

func mib(kib uint64) float64 { return round1(float64(kib) / 1024) }

// round1 rounds to one decimal - the precision top displays - so JSON
// output doesn't carry float noise like 0.40650406504065045.
func round1(v float64) float64 { return math.Round(v*10) / 10 }

// scaleKiB fits a KiB value into top's memory columns, switching to
// m/g/t suffixes when the plain number would overflow the width.
func scaleKiB(kib uint64, width int) string {
	s := strconv.FormatUint(kib, 10)
	if len(s) <= width {
		return s
	}
	v := float64(kib)
	for _, unit := range []string{"m", "g", "t"} {
		v /= 1024
		if out := strconv.FormatFloat(v, 'f', 1, 64) + unit; len(out) <= width {
			return out
		}
	}
	return strconv.FormatFloat(v, 'f', 0, 64) + "t"
}

// Format renders the snapshot like `top -b -n 1`.
func Format(s Snapshot) string { return format(s, false) }

// FormatWide adds PPID and thread-count columns and shows each process's
// full command line (kernel threads, which have none, as [name]) - like
// `top -c` with extra fields.
func FormatWide(s Snapshot) string { return format(s, true) }

func format(s Snapshot, wide bool) string {
	var b strings.Builder
	sm := s.Summary
	users := "users"
	if sm.Users == 1 {
		users = "user"
	}
	fmt.Fprintf(&b, "top - %s %s, %2d %s,  load average: %.2f, %.2f, %.2f\n",
		sm.Time, sm.Uptime, sm.Users, users, sm.LoadAverage[0], sm.LoadAverage[1], sm.LoadAverage[2])
	fmt.Fprintf(&b, "Tasks: %3d total, %3d running, %3d sleeping, %3d stopped, %3d zombie\n",
		sm.Tasks.Total, sm.Tasks.Running, sm.Tasks.Sleeping, sm.Tasks.Stopped, sm.Tasks.Zombie)
	c := sm.CPU
	fmt.Fprintf(&b, "%%Cpu(s): %4.1f us, %4.1f sy, %4.1f ni, %4.1f id, %4.1f wa, %4.1f hi, %4.1f si, %4.1f st\n",
		c.User, c.System, c.Nice, c.Idle, c.IOWait, c.IRQ, c.SoftIRQ, c.Steal)
	fmt.Fprintf(&b, "MiB Mem : %8.1f total, %8.1f free, %8.1f used, %8.1f buff/cache\n",
		sm.MemMiB.Total, sm.MemMiB.Free, sm.MemMiB.Used, sm.MemMiB.BuffCache)
	fmt.Fprintf(&b, "MiB Swap: %8.1f total, %8.1f free, %8.1f used. %8.1f avail Mem\n\n",
		sm.SwapMiB.Total, sm.SwapMiB.Free, sm.SwapMiB.Used, sm.SwapMiB.AvailMem)

	// top truncates USER to 8 columns ("privile+"); wide shows full names,
	// so size the column to the longest one to keep everything aligned.
	userW, extraHead := 8, ""
	if wide {
		extraHead = fmt.Sprintf(" %7s %4s", "PPID", "THR")
		for _, r := range s.Processes {
			userW = max(userW, len(r.User))
		}
	}
	fmt.Fprintf(&b, "%7s %-*s %3s %3s %7s %6s %6s %1s %5s %5s %9s%s %s\n",
		"PID", userW, "USER", "PR", "NI", "VIRT", "RES", "SHR", "S", "%CPU", "%MEM", "TIME+", extraHead, "COMMAND")
	for _, r := range s.Processes {
		user := r.User
		if len(user) > 8 && !wide {
			user = user[:7] + "+" // top's truncation marker
		}
		command, extra := r.Command, ""
		if wide {
			extra = fmt.Sprintf(" %7d %4d", r.PPID, r.Threads)
			command = r.Cmdline
			if command == "" {
				command = "[" + r.Command + "]" // kernel thread, as top -c shows it
			}
		}
		fmt.Fprintf(&b, "%7d %-*s %3s %3d %7s %6s %6s %1s %5.1f %5.1f %9s%s %s\n",
			r.PID, userW, user, r.PR, r.NI, scaleKiB(r.VirtKiB, 7), scaleKiB(r.ResKiB, 6), scaleKiB(r.ShrKiB, 6),
			r.State, r.CPUPercent, r.MemPercent, r.TimePlus, extra, command)
	}
	return b.String()
}
