//go:build windows

package routes

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"strings"
)

type windowsConfigurator struct{}

func newConfigurator() Configurator {
	return windowsConfigurator{}
}

func (windowsConfigurator) Apply(ctx context.Context, destination netip.Prefix) error {
	backend := windowsRouteBackend()
	gateway, err := backend.gateway(ctx)
	if err != nil {
		return err
	}

	if err := backend.apply(ctx, destination, gateway); err != nil {
		return err
	}

	return nil
}

type windowsBackend interface {
	gateway(ctx context.Context) (string, error)
	apply(ctx context.Context, destination netip.Prefix, gateway string) error
}

type windowsPowerShellBackend struct {
	binary string
}

type windowsCmdBackend struct{}

func windowsRouteBackend() windowsBackend {
	for _, candidate := range []string{"powershell.exe", "powershell", "pwsh.exe", "pwsh"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return windowsPowerShellBackend{binary: candidate}
		}
	}

	return windowsCmdBackend{}
}

func (b windowsPowerShellBackend) gateway(ctx context.Context) (string, error) {
	script := `(Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Sort-Object RouteMetric | Select-Object -First 1 -ExpandProperty NextHop)`
	output, err := exec.CommandContext(ctx, b.binary, "-NoProfile", "-Command", script).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("read default gateway via powershell: %w: %s", err, strings.TrimSpace(string(output)))
	}

	gateway := strings.TrimSpace(string(output))
	if gateway == "" {
		return "", fmt.Errorf("default gateway not found via powershell")
	}

	return gateway, nil
}

func (b windowsPowerShellBackend) apply(ctx context.Context, destination netip.Prefix, gateway string) error {
	network, mask := windowsRouteTarget(destination)
	script := fmt.Sprintf(
		`route DELETE %s MASK %s > $null 2>&1; route ADD %s MASK %s %s`,
		network,
		mask,
		network,
		mask,
		gateway,
	)
	output, err := exec.CommandContext(ctx, b.binary, "-NoProfile", "-Command", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("apply route %s via %s on windows powershell: %w: %s", destination, gateway, err, strings.TrimSpace(string(output)))
	}

	return nil
}

func (windowsCmdBackend) gateway(ctx context.Context) (string, error) {
	output, err := exec.CommandContext(ctx, "cmd", "/C", "route", "print", "-4", "0.0.0.0").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("read default gateway via cmd: %w: %s", err, strings.TrimSpace(string(output)))
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 4 && fields[0] == "0.0.0.0" && fields[1] == "0.0.0.0" {
			return fields[2], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("parse default gateway via cmd: %w", err)
	}

	return "", fmt.Errorf("default gateway not found via cmd")
}

func (windowsCmdBackend) apply(ctx context.Context, destination netip.Prefix, gateway string) error {
	network, mask := windowsRouteTarget(destination)
	_ = exec.CommandContext(ctx, "cmd", "/C", "route", "DELETE", network, "MASK", mask).Run()
	output, err := exec.CommandContext(ctx, "cmd", "/C", "route", "ADD", network, "MASK", mask, gateway).CombinedOutput()
	if err != nil {
		return fmt.Errorf("apply route %s via %s on windows cmd: %w: %s", destination, gateway, err, strings.TrimSpace(string(output)))
	}

	return nil
}

func windowsRouteTarget(destination netip.Prefix) (string, string) {
	masked := destination.Masked()
	mask := net.CIDRMask(masked.Bits(), masked.Addr().BitLen())

	return masked.Addr().String(), net.IP(mask).String()
}
