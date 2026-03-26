//go:build darwin

package routes

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/netip"
	"os/exec"
	"strings"
)

type darwinConfigurator struct{}

func newConfigurator() Configurator {
	return darwinConfigurator{}
}

func (darwinConfigurator) Apply(ctx context.Context, destination netip.Prefix) error {
	defaultIf, gateway, err := darwinDefaultRoute(ctx)
	if err != nil {
		return err
	}

	if strings.HasPrefix(defaultIf, "utun") {
		return fmt.Errorf("default route is currently %s; disconnect WireGuard/VPN first", defaultIf)
	}

	if gateway == "" {
		return fmt.Errorf("could not determine normal default gateway")
	}

	deleteArgs, addArgs := darwinRouteArgs(destination, gateway)
	_ = exec.CommandContext(ctx, "route", deleteArgs...).Run()

	output, err := exec.CommandContext(ctx, "route", addArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("apply route %s via %s on macos: %w: %s", destination, gateway, err, strings.TrimSpace(string(output)))
	}

	return nil
}

func darwinDefaultRoute(ctx context.Context) (string, string, error) {
	output, err := exec.CommandContext(ctx, "route", "-n", "get", "default").CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("read default route on macos: %w: %s", err, strings.TrimSpace(string(output)))
	}

	var defaultIf string
	var gateway string

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "interface:"):
			defaultIf = strings.TrimSpace(strings.TrimPrefix(line, "interface:"))
		case strings.HasPrefix(line, "gateway:"):
			gateway = strings.TrimSpace(strings.TrimPrefix(line, "gateway:"))
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("parse default route on macos: %w", err)
	}

	return defaultIf, gateway, nil
}

func darwinRouteArgs(destination netip.Prefix, gateway string) ([]string, []string) {
	if destination.Bits() == destination.Addr().BitLen() {
		host := destination.Addr().String()
		return []string{"-n", "delete", "-host", host}, []string{"-n", "add", "-host", host, gateway}
	}

	network := destination.Masked().String()
	return []string{"-n", "delete", "-net", network}, []string{"-n", "add", "-net", network, gateway}
}
