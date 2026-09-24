package performance

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type PerformanceArgs struct {
	Device       string `json:"device,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

type DeviceStat struct {
	Major             int    `json:"major" yaml:"major"`
	Minor             int    `json:"minor" yaml:"minor"`
	DeviceName        string `json:"device_name" yaml:"device_name"`
	ReadsCompleted    uint64 `json:"reads_completed" yaml:"reads_completed"`
	ReadsMerged       uint64 `json:"reads_merged" yaml:"reads_merged"`
	SectorsRead       uint64 `json:"sectors_read" yaml:"sectors_read"`
	TimeReadingMs     uint64 `json:"time_reading_ms" yaml:"time_reading_ms"`
	WritesCompleted   uint64 `json:"writes_completed" yaml:"writes_completed"`
	WritesMerged      uint64 `json:"writes_merged" yaml:"writes_merged"`
	SectorsWritten    uint64 `json:"sectors_written" yaml:"sectors_written"`
	TimeWritingMs     uint64 `json:"time_writing_ms" yaml:"time_writing_ms"`
	IosInProgress     uint64 `json:"ios_in_progress" yaml:"ios_in_progress"`
	TimeDoingIosMs    uint64 `json:"time_doing_ios_ms" yaml:"time_doing_ios_ms"`
	WeightedTimeIosMs uint64 `json:"weighted_time_ios_ms" yaml:"weighted_time_ios_ms"`
}

func Performance(args []byte) (string, error) {
	var params PerformanceArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	file, err := os.Open("/proc/diskstats")
	if err != nil {
		return "", fmt.Errorf("failed to open /proc/diskstats: %v", err)
	}
	defer file.Close()

	var stats []DeviceStat
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue // skip malformed lines or incomplete stats
		}

		devName := fields[2]

		// Filter by device if requested
		if params.Device != "" && devName != params.Device {
			continue
		}

		// Loop devices/partitions (e.g., loop0, ram0) are often noisy, filter them out unless explicitly requested
		if params.Device == "" && (strings.HasPrefix(devName, "loop") || strings.HasPrefix(devName, "ram")) {
			continue
		}

		major, _ := strconv.Atoi(fields[0])
		minor, _ := strconv.Atoi(fields[1])
		readsCompleted, _ := strconv.ParseUint(fields[3], 10, 64)
		readsMerged, _ := strconv.ParseUint(fields[4], 10, 64)
		sectorsRead, _ := strconv.ParseUint(fields[5], 10, 64)
		timeReading, _ := strconv.ParseUint(fields[6], 10, 64)
		writesCompleted, _ := strconv.ParseUint(fields[7], 10, 64)
		writesMerged, _ := strconv.ParseUint(fields[8], 10, 64)
		sectorsWritten, _ := strconv.ParseUint(fields[9], 10, 64)
		timeWriting, _ := strconv.ParseUint(fields[10], 10, 64)
		iosInProgress, _ := strconv.ParseUint(fields[11], 10, 64)
		timeDoingIos, _ := strconv.ParseUint(fields[12], 10, 64)
		weightedTimeIos, _ := strconv.ParseUint(fields[13], 10, 64)

		stat := DeviceStat{
			Major:             major,
			Minor:             minor,
			DeviceName:        devName,
			ReadsCompleted:    readsCompleted,
			ReadsMerged:       readsMerged,
			SectorsRead:       sectorsRead,
			TimeReadingMs:     timeReading,
			WritesCompleted:   writesCompleted,
			WritesMerged:      writesMerged,
			SectorsWritten:    sectorsWritten,
			TimeWritingMs:     timeWriting,
			IosInProgress:     iosInProgress,
			TimeDoingIosMs:    timeDoingIos,
			WeightedTimeIosMs: weightedTimeIos,
		}
		stats = append(stats, stat)
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading /proc/diskstats: %v", err)
	}
	if params.Device != "" && len(stats) == 0 {
		return "", fmt.Errorf("no such block device %q (see disks/list for device names)", params.Device)
	}

	if params.OutputFormat == "yaml" {
		out, err := yaml.Marshal(stats)
		if err != nil {
			return "", err
		}
		return string(out), nil
	} else if params.OutputFormat == "json" {
		out, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return "", err
		}
		return string(out), nil
	}

	// Default table/text format
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-12s %-12s %-12s %-12s %-12s %-12s\n", "Device", "Reads", "Writes", "SectRead", "SectWrite", "I/O(ms)"))
	sb.WriteString(strings.Repeat("-", 76) + "\n")
	for _, s := range stats {
		sb.WriteString(fmt.Sprintf("%-12s %-12d %-12d %-12d %-12d %-12d\n",
			s.DeviceName,
			s.ReadsCompleted,
			s.WritesCompleted,
			s.SectorsRead,
			s.SectorsWritten,
			s.TimeDoingIosMs,
		))
	}
	return sb.String(), nil
}
