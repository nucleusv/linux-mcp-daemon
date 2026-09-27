// Package networks lists Docker networks. Read-only by construction: no
// create, no remove, no connect/disconnect - removing a network is prune's
// job, and a container's membership is already in container://{name}/inspect.
package networks

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/nucleusv/linux-mcp-daemon/internal/docker"
)

type Args struct {
	docker.CommonArgs
	Pattern      string `json:"pattern,omitempty"`
	Driver       string `json:"driver,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

// apiNetwork is the subset of GET /networks this tool reports. The list
// endpoint carries an inspect's fields except Containers, which it leaves
// empty - the attached containers come from the container list instead, the
// way docker/volumes finds a volume's users.
type apiNetwork struct {
	Name       string `json:"Name"`
	ID         string `json:"Id"`
	Created    string `json:"Created"`
	Scope      string `json:"Scope"`
	Driver     string `json:"Driver"`
	EnableIPv6 bool   `json:"EnableIPv6"`
	Internal   bool   `json:"Internal"`
	Attachable bool   `json:"Attachable"`
	Ingress    bool   `json:"Ingress"`
	IPAM       struct {
		Driver string `json:"Driver"`
		Config []struct {
			Subnet  string `json:"Subnet"`
			Gateway string `json:"Gateway"`
			IPRange string `json:"IPRange"`
		} `json:"Config"`
	} `json:"IPAM"`
	Options map[string]string `json:"Options"`
	Labels  map[string]string `json:"Labels"`
}

func List(argsJSON []byte) (string, error) {
	var args Args
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	c := args.Client(30 * time.Second)
	var list []apiNetwork
	if err := c.GetJSON("/networks", &list); err != nil {
		return "", err
	}
	members := networkMembers(c)

	rows := make([]map[string]interface{}, 0, len(list))
	var text strings.Builder
	for _, n := range list {
		if args.Pattern != "" {
			if ok, _ := path.Match(args.Pattern, n.Name); !ok {
				continue
			}
		}
		if args.Driver != "" && !strings.EqualFold(n.Driver, args.Driver) {
			continue
		}
		subnets, gateways := ipam(n)
		if docker.Structured(args.OutputFormat) {
			rows = append(rows, map[string]interface{}{
				"name":        n.Name,
				"id":          short(n.ID),
				"full_id":     n.ID,
				"driver":      n.Driver,
				"scope":       n.Scope,
				"created":     n.Created,
				"ipv6":        n.EnableIPv6,
				"internal":    n.Internal,
				"attachable":  n.Attachable,
				"ingress":     n.Ingress,
				"ipam_driver": n.IPAM.Driver,
				"subnets":     subnets,
				"gateways":    gateways,
				"containers":  members[n.Name],
				"options":     n.Options,
				"labels":      n.Labels,
			})
			continue
		}
		fmt.Fprintf(&text, "%s (%s)\n  Driver: %s | Scope: %s%s\n", n.Name, short(n.ID), n.Driver, n.Scope, flags(n))
		if len(subnets) > 0 {
			fmt.Fprintf(&text, "  Subnet: %s | Gateway: %s\n", strings.Join(subnets, ", "), strings.Join(gateways, ", "))
		}
		if in := members[n.Name]; len(in) > 0 {
			fmt.Fprintf(&text, "  Attached: %s\n", strings.Join(in, ", "))
		} else {
			text.WriteString("  Attached: (nothing)\n")
		}
		text.WriteString("\n")
	}

	if docker.Structured(args.OutputFormat) {
		if len(rows) == 0 {
			return "[]", nil
		}
		b, _ := json.Marshal(rows)
		return string(b), nil
	}
	if text.Len() == 0 {
		return "No networks found matching the criteria.", nil
	}
	text.WriteString("Hint: For one network's IPAM, options and per-container addresses, read docker-network://<name>/inspect\n")
	return text.String(), nil
}

// ipam flattens the IPAM config blocks into parallel subnet and gateway lists.
// A network can have several of each (IPv4 plus IPv6, or several pools).
func ipam(n apiNetwork) (subnets, gateways []string) {
	for _, cfg := range n.IPAM.Config {
		if cfg.Subnet != "" {
			subnets = append(subnets, cfg.Subnet)
		}
		if cfg.Gateway != "" {
			gateways = append(gateways, cfg.Gateway)
		}
	}
	return subnets, gateways
}

// networkMembers maps each network name to the containers on it, as
// "name (address)".
//
// GET /networks leaves its Containers map empty, so the membership is read from
// the container list, which carries each container's networks and address -
// the same detour docker/volumes makes for a volume's users. Stopped
// containers are included and have no address; a failure here is not fatal,
// the listing is still worth returning without it.
func networkMembers(c *docker.Client) map[string][]string {
	var list []struct {
		Names           []string `json:"Names"`
		NetworkSettings struct {
			Networks map[string]struct {
				IPAddress         string `json:"IPAddress"`
				GlobalIPv6Address string `json:"GlobalIPv6Address"`
			} `json:"Networks"`
		} `json:"NetworkSettings"`
	}
	if err := c.GetJSON("/containers/json"+docker.Q("all", "true"), &list); err != nil {
		return nil
	}
	members := map[string][]string{}
	for _, ct := range list {
		name := ""
		if len(ct.Names) > 0 {
			name = strings.TrimPrefix(ct.Names[0], "/")
		}
		for net, ep := range ct.NetworkSettings.Networks {
			addr := ep.IPAddress
			if addr == "" {
				addr = ep.GlobalIPv6Address
			}
			if addr == "" {
				members[net] = append(members[net], name)
				continue
			}
			members[net] = append(members[net], fmt.Sprintf("%s (%s)", name, addr))
		}
	}
	for k := range members {
		sort.Strings(members[k])
	}
	return members
}

// flags reports only the flags that are set - "Internal: false" on every one of
// a host's networks is noise.
func flags(n apiNetwork) string {
	var on []string
	if n.Internal {
		on = append(on, "internal")
	}
	if n.Attachable {
		on = append(on, "attachable")
	}
	if n.Ingress {
		on = append(on, "ingress")
	}
	if n.EnableIPv6 {
		on = append(on, "ipv6")
	}
	if len(on) == 0 {
		return ""
	}
	return " | " + strings.Join(on, ", ")
}

func short(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
