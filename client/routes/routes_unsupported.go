//go:build !linux && !darwin && !windows

package routes

import (
	"context"
	"fmt"
	"net/netip"
)

type unsupportedConfigurator struct{}

func newConfigurator() Configurator {
	return unsupportedConfigurator{}
}

func (unsupportedConfigurator) Apply(_ context.Context, destination netip.Prefix) error {
	return fmt.Errorf("route configuration is not supported on this OS for %s", destination)
}
