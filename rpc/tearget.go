package rpc

import (
	"fmt"
	"strings"
)

// BuildDirectTarget returns a string that represents the given endpoints with direct schema.
func BuildDirectTarget(endpoints []string) string {
	return fmt.Sprintf("%s:///%s", DirectScheme,
		strings.Join(endpoints, ","))
}

// BuildDiscovTarget returns a string that represents the given endpoints with discov schema.
func BuildDiscoverTarget(endpoints []string, key string) string {
	return fmt.Sprintf("%s://%s/%s", DiscoverScheme,
		strings.Join(endpoints, ","), key)
}
