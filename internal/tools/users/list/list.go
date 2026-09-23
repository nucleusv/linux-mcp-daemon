package list

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// GetUsersArgs are the tool's input arguments.
type GetUsersArgs struct {
	OutputFormat string `json:"output_format,omitempty"` // Desired output format (json/yaml/table/wide). Defaults to text.
	MinUID       int    `json:"min_uid,omitempty"`        // Only include users with UID >= this value (e.g. 1000 to exclude system accounts).
	Privileged   bool   `json:"privileged,omitempty"`     // Run as root.
}

// User describes one entry from /etc/passwd, augmented with group info.
type User struct {
	Username  string   `json:"username"`
	UID       int      `json:"uid"`
	GID       int      `json:"gid"`
	GroupName string   `json:"group_name,omitempty"`
	Comment   string   `json:"comment,omitempty"` // The GECOS field - typically the user's full name.
	HomeDir   string   `json:"home_dir"`
	Shell     string   `json:"shell"`
	Groups    []string `json:"groups,omitempty"` // Supplementary group memberships, from /etc/group.
}

// List returns the users defined on this system, natively parsing
// /etc/passwd and /etc/group. Deliberately never reads /etc/shadow - this
// tool reports account identity/metadata, not credentials.
func List(argsJSON []byte) (string, error) {
	var args GetUsersArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %v", err)
		}
	}

	groupNames, memberGroups, err := parseGroups("/etc/group")
	if err != nil {
		return "", err
	}

	f, err := os.Open("/etc/passwd")
	if err != nil {
		return "", fmt.Errorf("failed to open /etc/passwd: %v", err)
	}
	defer f.Close()

	var users []User
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Format: username:password:uid:gid:gecos:home:shell
		fields := strings.Split(line, ":")
		if len(fields) < 7 {
			continue
		}
		uid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		if uid < args.MinUID {
			continue
		}
		gid, _ := strconv.Atoi(fields[3])
		users = append(users, User{
			Username:  fields[0],
			UID:       uid,
			GID:       gid,
			GroupName: groupNames[gid],
			Comment:   fields[4],
			HomeDir:   fields[5],
			Shell:     fields[6],
			Groups:    memberGroups[fields[0]],
		})
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read /etc/passwd: %v", err)
	}

	sort.Slice(users, func(i, j int) bool { return users[i].UID < users[j].UID })

	if args.OutputFormat == "json" || args.OutputFormat == "yaml" || args.OutputFormat == "table" || args.OutputFormat == "wide" {
		b, err := json.Marshal(users)
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %v", err)
		}
		return string(b), nil
	}

	var sb strings.Builder
	for _, u := range users {
		primary := u.GroupName
		if primary == "" {
			primary = strconv.Itoa(u.GID)
		}
		line := fmt.Sprintf("%s (uid=%d gid=%d(%s)) home=%s shell=%s", u.Username, u.UID, u.GID, primary, u.HomeDir, u.Shell)
		if len(u.Groups) > 0 {
			line += " groups=" + strings.Join(u.Groups, ",")
		}
		sb.WriteString(line + "\n")
	}
	return sb.String(), nil
}

// parseGroups reads /etc/group, returning a gid->name map (for resolving a
// user's primary group) and a username->[]groupname map (for supplementary
// memberships, from each group's comma-separated member list).
func parseGroups(path string) (map[int]string, map[string][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open %s: %v", path, err)
	}
	defer f.Close()

	groupNames := make(map[int]string)
	memberGroups := make(map[string][]string)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Format: groupname:password:gid:member1,member2,...
		fields := strings.Split(line, ":")
		if len(fields) < 4 {
			continue
		}
		name := fields[0]
		gid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		groupNames[gid] = name
		if fields[3] != "" {
			for _, member := range strings.Split(fields[3], ",") {
				memberGroups[member] = append(memberGroups[member], name)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to read %s: %v", path, err)
	}
	return groupNames, memberGroups, nil
}
