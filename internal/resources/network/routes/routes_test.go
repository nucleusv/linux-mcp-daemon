package routes

import (
	"reflect"
	"testing"
)

// Lines captured from the Ubuntu test VPS's /proc/net/route.
func TestParseRoute(t *testing.T) {
	cases := []struct {
		line string
		want Route
	}{
		{
			"ens1\t00000000\t01D4A7DE\t0003\t0\t0\t100\t00000000\t0\t0\t0",
			Route{Destination: "0.0.0.0/0", Gateway: "222.167.212.1", Iface: "ens1", Metric: 100, Flags: []string{"up", "gateway"}, Default: true},
		},
		{
			"docker0\t000011AC\t00000000\t0001\t0\t0\t0\t0000FFFF\t0\t0\t0",
			Route{Destination: "172.17.0.0/16", Iface: "docker0", Flags: []string{"up"}},
		},
		{
			"ens1\t00D4A7DE\t00000000\t0001\t0\t0\t100\t00FFFFFF\t0\t0\t0",
			Route{Destination: "222.167.212.0/24", Iface: "ens1", Metric: 100, Flags: []string{"up"}},
		},
	}
	for _, c := range cases {
		got, ok := parseRoute(c.line)
		if !ok || !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseRoute(%q)\n got  %+v\n want %+v", c.line, got, c.want)
		}
	}
	if _, ok := parseRoute("garbage"); ok {
		t.Error("parsed a malformed line")
	}
}
