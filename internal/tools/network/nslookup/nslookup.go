package nslookup

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

// NslookupArgs defines the parameters for the network/nslookup tool.
type NslookupArgs struct {
	Host         string `json:"host"`                    // Host is the hostname or IP to resolve. Required.
	RecordType   string `json:"record_type,omitempty"`   // RecordType specifies the DNS record type (e.g. A, TXT, MX, CNAME, NS). Defaults to ANY or standard A/AAAA.
	OutputFormat string `json:"output_format,omitempty"` // OutputFormat specifies the desired output format. Defaults to text.
}

type Record struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type NslookupResponse struct {
	Host    string   `json:"host"`
	Records []Record `json:"records"`
}

func Nslookup(argsJSON []byte) (string, error) {
	var args NslookupArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	args.Host = strings.TrimSpace(args.Host)
	if args.Host == "" {
		return "", fmt.Errorf("host argument is required")
	}

	rtype := strings.ToUpper(strings.TrimSpace(args.RecordType))
	if rtype == "" {
		rtype = "ANY"
	}

	var results []Record

	// CNAME
	if rtype == "CNAME" || rtype == "ANY" {
		if cname, err := net.LookupCNAME(args.Host); err == nil && cname != "" {
			results = append(results, Record{Type: "CNAME", Value: cname})
		}
	}

	// A / AAAA (IPs)
	if rtype == "A" || rtype == "AAAA" || rtype == "ANY" {
		if ips, err := net.LookupIP(args.Host); err == nil {
			for _, ip := range ips {
				recordType := "A"
				if ip.To4() == nil {
					recordType = "AAAA"
				}
				if rtype == "ANY" || rtype == recordType {
					results = append(results, Record{Type: recordType, Value: ip.String()})
				}
			}
		}
	}

	// TXT
	if rtype == "TXT" || rtype == "ANY" {
		if txts, err := net.LookupTXT(args.Host); err == nil {
			for _, txt := range txts {
				results = append(results, Record{Type: "TXT", Value: txt})
			}
		}
	}

	// MX
	if rtype == "MX" || rtype == "ANY" {
		if mxs, err := net.LookupMX(args.Host); err == nil {
			for _, mx := range mxs {
				results = append(results, Record{Type: "MX", Value: fmt.Sprintf("%d %s", mx.Pref, mx.Host)})
			}
		}
	}

	// NS
	if rtype == "NS" || rtype == "ANY" {
		if nss, err := net.LookupNS(args.Host); err == nil {
			for _, ns := range nss {
				results = append(results, Record{Type: "NS", Value: ns.Host})
			}
		}
	}

	response := NslookupResponse{
		Host:    args.Host,
		Records: results,
	}

	out, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode response: %v", err)
	}

	return string(out), nil
}
