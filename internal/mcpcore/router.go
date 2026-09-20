package mcpcore

import (
	"golang.org/x/sync/singleflight"
)

type Router struct {
	requestGroup singleflight.Group
}
// Execute logic with SingleFlight