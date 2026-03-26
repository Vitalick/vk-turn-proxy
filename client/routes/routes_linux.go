//go:build linux

package routes

import (
	"bytes"
	"context"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"strings"
)

type linuxConfigurator struct{}

func newConfigurator() Configurator {
	return linuxConfigurator{}
}

func (linuxConfigurator) CheckPrivileges(_ context.Context) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("insufficient privileges on linux: run the client as root (for example via sudo)")
	}

	return nil
}

func (linuxConfigurator) Apply(ctx context.Context, destination netip.Prefix) error {
	gateway, err := linuxGateway(ctx)
	if err != nil {
		return err
	}

	args := []string{"route", "replace", destination.String(), "via", gateway}
	output, err := exec.CommandContext(ctx, "ip", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("apply route %s via %s on linux: %w: %s", destination, gateway, err, strings.TrimSpace(string(output)))
	}

	return nil
}

func linuxGateway(ctx context.Context) (string, error) {
	output, err := exec.CommandContext(ctx, "ip", "-o", "-4", "route", "show", "to", "default").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("read default ipv4 route on linux: %w: %s", err, strings.TrimSpace(string(output)))
	}

	fields := bytes.Fields(output)
	for idx := 0; idx < len(fields)-1; idx++ {
		if string(fields[idx]) == "via" {
			return string(fields[idx+1]), nil
		}
	}

	return "", fmt.Errorf("default ipv4 gateway not found in %q", strings.TrimSpace(string(output)))
}
