package main

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"roller_hoops/core-go/internal/db"
	"roller_hoops/core-go/internal/discoveryworker"
	"roller_hoops/core-go/internal/httpapi"
	"roller_hoops/core-go/internal/metrics"
)

func main() {
	addr := envOr("HTTP_ADDR", ":8081")
	logLevel := envOr("LOG_LEVEL", "info")
	databaseURL := envOr("DATABASE_URL", "")

	logger := httpapi.NewLogger(logLevel)
	sharedMetrics := metrics.New()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var pool *db.Pool
	if databaseURL != "" {
		p, err := db.Open(ctx, databaseURL)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to connect to database")
		}
		defer p.Close()
		pool = p
	}

	if pool != nil {
		opts := discoveryworker.Options{
			PollInterval:          envOrDuration("DISCOVERY_POLL_INTERVAL", 400*time.Millisecond),
			RunDelay:              envOrDuration("DISCOVERY_RUN_DELAY", 0),
			MaxRuntime:            envOrDuration("DISCOVERY_MAX_RUNTIME", 30*time.Second),
			ARPTablePath:          envOr("DISCOVERY_ARP_TABLE_PATH", "/proc/net/arp"),
			MaxTargets:            envOrInt("DISCOVERY_MAX_TARGETS", 1024),
			PingTimeout:           envOrDuration("DISCOVERY_PING_TIMEOUT", 800*time.Millisecond),
			PingWorkers:           envOrInt("DISCOVERY_PING_WORKERS", 16),
			EnrichMaxTargets:      envOrInt("DISCOVERY_ENRICH_MAX_TARGETS", 64),
			EnrichWorkers:         envOrInt("DISCOVERY_ENRICH_WORKERS", 8),
			NameResolutionEnabled: envOrBool("DISCOVERY_NAME_RESOLUTION_ENABLED", true),
			SNMPEnabled:           envOrBool("DISCOVERY_SNMP_ENABLED", false),
			SNMPCommunity:         envOr("DISCOVERY_SNMP_COMMUNITY", "public"),
			SNMPVersion:           envOr("DISCOVERY_SNMP_VERSION", "2c"),
			SNMPTimeout:           envOrDuration("DISCOVERY_SNMP_TIMEOUT", 900*time.Millisecond),
			SNMPRetries:           envOrInt("DISCOVERY_SNMP_RETRIES", 0),
			SNMPPort:              uint16(envOrInt("DISCOVERY_SNMP_PORT", 161)),
			TopologyLLDPEnabled:   envOrBool("DISCOVERY_TOPOLOGY_LLDP_ENABLED", false),
			TopologyCDPEnabled:    envOrBool("DISCOVERY_TOPOLOGY_CDP_ENABLED", false),
			TopologyAllowlist:     envOrPrefixList("DISCOVERY_TOPOLOGY_ALLOWLIST"),
			PortScanEnabled:       envOrBool("DISCOVERY_PORT_SCAN_ENABLED", false),
			PortScanAllowlist:     envOrPrefixList("DISCOVERY_PORT_SCAN_ALLOWLIST"),
			PortScanPorts:         envOrPortList("DISCOVERY_PORT_SCAN_PORTS", []int{22, 80, 443}),
			PortScanWorkers:       envOrInt("DISCOVERY_PORT_SCAN_WORKERS", 4),
			PortScanTimeout:       envOrDuration("DISCOVERY_PORT_SCAN_TIMEOUT", 3*time.Second),
			PortScanMaxTargets:    envOrInt("DISCOVERY_PORT_SCAN_MAX_TARGETS", 24),
		}
		worker := discoveryworker.New(logger, pool.Queries(), opts, sharedMetrics)
		go worker.Run(ctx)
	}

	defaultDiscoveryScope, err := parseDiscoveryDefaultScope(envOr("DISCOVERY_DEFAULT_SCOPE", ""))
	if err != nil {
		logger.Fatal().Err(err).Msg("invalid DISCOVERY_DEFAULT_SCOPE")
	}
	h := httpapi.NewHandlerWithOptions(logger, pool, sharedMetrics, httpapi.Options{
		DiscoveryDefaultScope: defaultDiscoveryScope,
	})
	srv := &http.Server{
		Addr:              addr,
		Handler:           h.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info().Str("addr", addr).Msg("core-go listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("http server error")
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Info().Msg("shutdown complete")
}

func envOr(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func envOrInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envOrDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func envOrBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func envOrPrefixList(key string) []netip.Prefix {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]netip.Prefix, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		pfx, err := netip.ParsePrefix(s)
		if err != nil {
			continue
		}
		out = append(out, pfx.Masked())
	}
	return out
}

func envOrPortList(key string, fallback []int) []int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parts := strings.Split(raw, ",")
	seen := map[int]struct{}{}
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > 65535 {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	sort.Ints(out)
	if len(out) == 0 {
		return fallback
	}
	return out
}

func parseDiscoveryDefaultScope(raw string) (*string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, nil
	}

	if p, err := netip.ParsePrefix(s); err == nil {
		c := p.String()
		return &c, nil
	}
	if a, err := netip.ParseAddr(s); err == nil {
		c := a.String()
		return &c, nil
	}

	return nil, fmt.Errorf("must be a CIDR prefix or a single IP (got %q)", s)
}
