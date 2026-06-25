package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/miekg/dns"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/config"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/dnsserver"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/failover"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/health"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/runtimecfg"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/status"
	"github.com/nikitadybov/owrt-failsafe-dns-proxy-go/internal/upstream"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "failsafe-dns-proxy:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError()
	}
	switch args[0] {
	case "run":
		return runDaemon(args[1:])
	case "check-config":
		return checkConfig(args[1:])
	case "status":
		return showStatus(args[1:])
	case "probe":
		return probeUpstream(args[1:])
	case "self-test":
		return selfTest(args[1:])
	case "version":
		fmt.Println(version)
		return nil
	default:
		return usageError()
	}
}

func runDaemon(args []string) error {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	path := flags.String("config", config.DefaultPath, "path to UCI config")
	debug := flags.Bool("debug", false, "enable debug logging")
	if err := flags.Parse(args); err != nil {
		return err
	}
	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	manager := failover.NewManager(cfg.Upstreams, cfg.FailThreshold, cfg.RecoverThreshold, logger)
	exchanger := upstream.DNSExchanger{}
	store := runtimecfg.New(cfg, manager)
	server := dnsserver.New(store, exchanger, logger)
	statusServer := status.New(cfg.StatusSocket, version, store)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopSignals := make(chan os.Signal, 1)
	reloadSignals := make(chan os.Signal, 1)
	signal.Notify(stopSignals, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(reloadSignals, syscall.SIGHUP)
	defer signal.Stop(stopSignals)
	defer signal.Stop(reloadSignals)

	probeCtx, stopProbes := context.WithCancel(ctx)
	go health.New(cfg, manager, exchanger, logger).Run(probeCtx)
	errs := make(chan error, 2)
	go func() { errs <- statusServer.Run(ctx) }()
	go func() { errs <- server.Run(ctx) }()

	for {
		select {
		case <-stopSignals:
			stopProbes()
			cancel()
			return nil
		case <-reloadSignals:
			next, loadErr := config.Load(*path)
			if loadErr == nil {
				loadErr = runtimecfg.ValidateReload(store.Load().Config, next)
			}
			if loadErr != nil {
				store.RecordReloadError(loadErr, time.Now())
				logger.Error("configuration reload rejected", "error", loadErr)
				continue
			}
			previous := store.Load().Manager.Snapshots()
			nextManager := failover.NewManagerFromSnapshots(
				next.Upstreams, next.FailThreshold, next.RecoverThreshold, logger, previous,
			)
			store.Update(next, nextManager, time.Now())
			stopProbes()
			probeCtx, stopProbes = context.WithCancel(ctx)
			go health.New(next, nextManager, exchanger, logger).Run(probeCtx)
			logger.Info("configuration reloaded", "upstreams", len(next.Upstreams))
		case err := <-errs:
			stopProbes()
			cancel()
			if err == nil || errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
	}
}

func selfTest(args []string) error {
	flags := flag.NewFlagSet("self-test", flag.ContinueOnError)
	path := flags.String("config", config.DefaultPath, "path to UCI config")
	asJSON := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}
	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	probe := cfg.Probes[0]
	request := new(dns.Msg)
	request.SetQuestion(probe.Name, probe.QType)
	loopback := config.Upstream{
		Name:     "local-proxy",
		Enabled:  true,
		Priority: 1,
		Protocol: "udp",
		Address:  cfg.ListenAddr,
		Port:     cfg.ListenPort,
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	defer cancel()
	response, latency, err := (upstream.DNSExchanger{}).Exchange(ctx, loopback, request)
	result := struct {
		Success   bool   `json:"success"`
		LatencyMS int64  `json:"latency_ms,omitempty"`
		Rcode     string `json:"rcode,omitempty"`
		Error     string `json:"error,omitempty"`
	}{}
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Success = response.Rcode != dns.RcodeServerFailure && response.Rcode != dns.RcodeRefused
		result.LatencyMS = latency.Milliseconds()
		result.Rcode = dns.RcodeToString[response.Rcode]
		if !result.Success {
			result.Error = "proxy returned " + result.Rcode
		}
	}
	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(result)
	}
	if !result.Success {
		return errors.New(result.Error)
	}
	fmt.Printf("local proxy: %s in %d ms\n", result.Rcode, result.LatencyMS)
	return nil
}

func checkConfig(args []string) error {
	flags := flag.NewFlagSet("check-config", flag.ContinueOnError)
	path := flags.String("config", config.DefaultPath, "path to UCI config")
	if err := flags.Parse(args); err != nil {
		return err
	}
	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	fmt.Printf("configuration valid: %d upstream(s), listen %s:%d\n", len(cfg.Upstreams), cfg.ListenAddr, cfg.ListenPort)
	return nil
}

func showStatus(args []string) error {
	flags := flag.NewFlagSet("status", flag.ContinueOnError)
	path := flags.String("socket", "/var/run/failsafe-dns-proxy.sock", "status socket")
	asJSON := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}
	doc, err := status.Read(*path, time.Second)
	if err != nil {
		return err
	}
	if *asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(doc)
	}
	fmt.Printf("active: %s\n", doc.Active)
	for _, item := range doc.Upstreams {
		fmt.Printf("%s priority=%d state=%s failures=%d successes=%d\n",
			item.Name, item.Priority, item.State, item.ConsecutiveFailures, item.ConsecutiveSuccess)
	}
	return nil
}

func probeUpstream(args []string) error {
	flags := flag.NewFlagSet("probe", flag.ContinueOnError)
	path := flags.String("config", config.DefaultPath, "path to UCI config")
	asJSON := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("probe requires one upstream name")
	}
	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	var selected *config.Upstream
	for i := range cfg.Upstreams {
		if cfg.Upstreams[i].Name == flags.Arg(0) {
			selected = &cfg.Upstreams[i]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("unknown upstream %q", flags.Arg(0))
	}
	probe := cfg.Probes[0]
	request := new(dns.Msg)
	request.SetQuestion(probe.Name, probe.QType)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.AttemptTimeout)
	defer cancel()
	response, latency, err := (upstream.DNSExchanger{}).Exchange(ctx, *selected, request)
	result := struct {
		Upstream  string `json:"upstream"`
		Success   bool   `json:"success"`
		LatencyMS int64  `json:"latency_ms,omitempty"`
		Rcode     string `json:"rcode,omitempty"`
		Error     string `json:"error,omitempty"`
	}{
		Upstream: selected.Name,
	}
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Success = response.Rcode != dns.RcodeServerFailure && response.Rcode != dns.RcodeRefused
		result.LatencyMS = latency.Milliseconds()
		result.Rcode = dns.RcodeToString[response.Rcode]
		if !result.Success {
			result.Error = "upstream returned " + result.Rcode
		}
	}
	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(result)
	}
	if !result.Success {
		return errors.New(result.Error)
	}
	fmt.Printf("%s: %s in %d ms\n", result.Upstream, result.Rcode, result.LatencyMS)
	return nil
}

func usageError() error {
	return errors.New("usage: failsafe-dns-proxy {run|check-config|status|probe|self-test|version}")
}
