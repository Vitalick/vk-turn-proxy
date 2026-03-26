package routes

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
	"sync"
)

type Configurator interface {
	Apply(ctx context.Context, destination netip.Prefix) error
}

type routeState struct {
	once sync.Once
	err  error
}

var applied sync.Map

func Apply(ctx context.Context, rawDestination string) error {
	destination, err := normalizeDestination(rawDestination)
	if err != nil {
		return err
	}

	stateAny, _ := applied.LoadOrStore(destination.String(), &routeState{})
	state := stateAny.(*routeState)
	state.once.Do(func() {
		state.err = newConfigurator().Apply(ctx, destination)
	})

	return state.err
}

func normalizeDestination(raw string) (netip.Prefix, error) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return netip.Prefix{}, fmt.Errorf("empty route destination")
	}

	if strings.Contains(candidate, "/") {
		prefix, err := netip.ParsePrefix(candidate)
		if err != nil {
			return netip.Prefix{}, fmt.Errorf("parse route prefix %q: %w", candidate, err)
		}

		return prefix.Masked(), nil
	}

	addr, err := netip.ParseAddr(candidate)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("parse route host %q: %w", candidate, err)
	}

	return netip.PrefixFrom(addr, addr.BitLen()), nil
}
