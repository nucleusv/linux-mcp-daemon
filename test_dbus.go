package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"
)

func main() {
	conn, err := dbus.NewSystemdConnectionContext(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer conn.Close()

	units, err := conn.ListUnitsContext(context.Background())
	if err != nil {
		fmt.Printf("Error listing units: %v\n", err)
		return
	}

	for _, u := range units {
		if strings.HasSuffix(u.Name, ".service") {
			fmt.Printf("%s | Load: %s | Active: %s | Sub: %s\n", u.Name, u.LoadState, u.ActiveState, u.SubState)
		}
	}
}
