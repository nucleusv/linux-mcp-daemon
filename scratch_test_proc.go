package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"io/ioutil"
)

type Process struct {
	PID     int    `json:"pid"`
	User    string `json:"user"`
	Comm    string `json:"comm"`
	State   string `json:"state"`
	PPID    int    `json:"ppid"`
	RSS     int64  `json:"rss_kb"`
	Cmdline string `json:"cmdline"`
}

func main() {
	entries, _ := os.ReadDir("/proc")
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
		
		procs = append(procs, p)
		if len(procs) >= 5 {
			break
		}
	}
	j, _ := json.MarshalIndent(procs, "", "  ")
	fmt.Println(string(j))
}
