package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const DefaultPath = "/etc/config/failsafe-dns-proxy"

type Config struct {
	Enabled          bool
	ListenAddr       string
	ListenPort       int
	AttemptTimeout   time.Duration
	RequestTimeout   time.Duration
	HealthInterval   time.Duration
	FailThreshold    int
	RecoverThreshold int
	MaxConcurrent    int
	StatusSocket     string
	Probes           []Probe
	Upstreams        []Upstream
}

type Probe struct {
	Name  string
	QType uint16
}

type Upstream struct {
	Name     string
	Enabled  bool
	Priority int
	Protocol string
	Address  string
	Port     int
}

func (u Upstream) Endpoint() string {
	return net.JoinHostPort(u.Address, strconv.Itoa(u.Port))
}

func Defaults() Config {
	return Config{
		Enabled:          true,
		ListenAddr:       "127.0.0.1",
		ListenPort:       5359,
		AttemptTimeout:   700 * time.Millisecond,
		RequestTimeout:   2 * time.Second,
		HealthInterval:   5 * time.Second,
		FailThreshold:    2,
		RecoverThreshold: 2,
		MaxConcurrent:    128,
		StatusSocket:     "/var/run/failsafe-dns-proxy.sock",
	}
}

func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config %q: %w", path, err)
	}
	defer f.Close()
	return Parse(f)
}

type section struct {
	kind    string
	name    string
	options map[string]string
	lists   map[string][]string
	line    int
}

func Parse(r io.Reader) (Config, error) {
	cfg := Defaults()
	var sections []section
	var current *section
	scanner := bufio.NewScanner(r)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		fields, err := splitUCI(line)
		if err != nil {
			return Config{}, fmt.Errorf("line %d: %w", lineNo, err)
		}
		switch fields[0] {
		case "config":
			if len(fields) < 2 || len(fields) > 3 {
				return Config{}, fmt.Errorf("line %d: config expects type and optional name", lineNo)
			}
			s := section{kind: fields[1], options: map[string]string{}, lists: map[string][]string{}, line: lineNo}
			if len(fields) == 3 {
				s.name = fields[2]
			}
			sections = append(sections, s)
			current = &sections[len(sections)-1]
		case "option", "list":
			if current == nil {
				return Config{}, fmt.Errorf("line %d: %s outside section", lineNo, fields[0])
			}
			if len(fields) != 3 {
				return Config{}, fmt.Errorf("line %d: %s expects key and value", lineNo, fields[0])
			}
			if fields[0] == "option" {
				current.options[fields[1]] = fields[2]
			} else {
				current.lists[fields[1]] = append(current.lists[fields[1]], fields[2])
			}
		default:
			return Config{}, fmt.Errorf("line %d: unsupported directive %q", lineNo, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var haveMain bool
	for _, s := range sections {
		switch s.kind {
		case "main":
			if haveMain {
				return Config{}, errors.New("multiple main sections")
			}
			haveMain = true
			if err := applyMain(&cfg, s); err != nil {
				return Config{}, err
			}
		case "upstream":
			u, err := parseUpstream(s)
			if err != nil {
				return Config{}, err
			}
			if u.Enabled {
				cfg.Upstreams = append(cfg.Upstreams, u)
			}
		default:
			return Config{}, fmt.Errorf("line %d: unsupported section type %q", s.line, s.kind)
		}
	}
	if !haveMain {
		return Config{}, errors.New("missing main section")
	}
	sort.SliceStable(cfg.Upstreams, func(i, j int) bool {
		if cfg.Upstreams[i].Priority == cfg.Upstreams[j].Priority {
			return cfg.Upstreams[i].Name < cfg.Upstreams[j].Name
		}
		return cfg.Upstreams[i].Priority < cfg.Upstreams[j].Priority
	})
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyMain(cfg *Config, s section) error {
	var err error
	if cfg.Enabled, err = boolOption(s, "enabled", cfg.Enabled); err != nil {
		return err
	}
	cfg.ListenAddr = stringOption(s, "listen_addr", cfg.ListenAddr)
	cfg.StatusSocket = stringOption(s, "status_socket", cfg.StatusSocket)
	if cfg.ListenPort, err = intOption(s, "listen_port", cfg.ListenPort); err != nil {
		return err
	}
	var ms int
	if ms, err = intOption(s, "attempt_timeout_ms", int(cfg.AttemptTimeout/time.Millisecond)); err != nil {
		return err
	}
	cfg.AttemptTimeout = time.Duration(ms) * time.Millisecond
	if ms, err = intOption(s, "request_timeout_ms", int(cfg.RequestTimeout/time.Millisecond)); err != nil {
		return err
	}
	cfg.RequestTimeout = time.Duration(ms) * time.Millisecond
	var seconds int
	if seconds, err = intOption(s, "health_interval_s", int(cfg.HealthInterval/time.Second)); err != nil {
		return err
	}
	cfg.HealthInterval = time.Duration(seconds) * time.Second
	if cfg.FailThreshold, err = intOption(s, "fail_threshold", cfg.FailThreshold); err != nil {
		return err
	}
	if cfg.RecoverThreshold, err = intOption(s, "recover_threshold", cfg.RecoverThreshold); err != nil {
		return err
	}
	if cfg.MaxConcurrent, err = intOption(s, "max_concurrent", cfg.MaxConcurrent); err != nil {
		return err
	}
	for _, raw := range s.lists["probe"] {
		probe, parseErr := parseProbe(raw)
		if parseErr != nil {
			return fmt.Errorf("main probe %q: %w", raw, parseErr)
		}
		cfg.Probes = append(cfg.Probes, probe)
	}
	return nil
}

func parseUpstream(s section) (Upstream, error) {
	u := Upstream{Name: s.name, Enabled: true, Protocol: "udp", Port: 53}
	var err error
	if u.Name == "" {
		return Upstream{}, fmt.Errorf("line %d: upstream requires a name", s.line)
	}
	if u.Enabled, err = boolOption(s, "enabled", true); err != nil {
		return Upstream{}, fmt.Errorf("upstream %q: %w", u.Name, err)
	}
	if u.Priority, err = intOption(s, "priority", 0); err != nil {
		return Upstream{}, fmt.Errorf("upstream %q: %w", u.Name, err)
	}
	u.Protocol = strings.ToLower(stringOption(s, "protocol", u.Protocol))
	u.Address = stringOption(s, "address", "")
	if u.Port, err = intOption(s, "port", u.Port); err != nil {
		return Upstream{}, fmt.Errorf("upstream %q: %w", u.Name, err)
	}
	return u, nil
}

func (c Config) Validate() error {
	var problems []string
	if net.ParseIP(c.ListenAddr) == nil {
		problems = append(problems, "listen_addr must be an IP address")
	}
	if c.ListenPort < 1 || c.ListenPort > 65535 {
		problems = append(problems, "listen_port must be between 1 and 65535")
	}
	if c.AttemptTimeout <= 0 {
		problems = append(problems, "attempt_timeout_ms must be positive")
	}
	if c.RequestTimeout < c.AttemptTimeout {
		problems = append(problems, "request_timeout_ms must be >= attempt_timeout_ms")
	}
	if c.HealthInterval <= 0 {
		problems = append(problems, "health_interval_s must be positive")
	}
	if c.FailThreshold < 1 || c.RecoverThreshold < 1 {
		problems = append(problems, "failure and recovery thresholds must be positive")
	}
	if c.MaxConcurrent < 1 {
		problems = append(problems, "max_concurrent must be positive")
	}
	if c.StatusSocket == "" {
		problems = append(problems, "status_socket must not be empty")
	}
	if len(c.Probes) == 0 {
		problems = append(problems, "at least one probe is required")
	}
	if len(c.Upstreams) == 0 {
		problems = append(problems, "at least one enabled upstream is required")
	}
	names := map[string]bool{}
	priorities := map[int]bool{}
	for _, u := range c.Upstreams {
		if names[u.Name] {
			problems = append(problems, fmt.Sprintf("duplicate upstream name %q", u.Name))
		}
		names[u.Name] = true
		if priorities[u.Priority] {
			problems = append(problems, fmt.Sprintf("duplicate upstream priority %d", u.Priority))
		}
		priorities[u.Priority] = true
		if u.Priority < 0 {
			problems = append(problems, fmt.Sprintf("upstream %q priority must be non-negative", u.Name))
		}
		if u.Protocol != "udp" && u.Protocol != "tcp" {
			problems = append(problems, fmt.Sprintf("upstream %q protocol must be udp or tcp", u.Name))
		}
		if net.ParseIP(u.Address) == nil {
			problems = append(problems, fmt.Sprintf("upstream %q address must be an IP address", u.Name))
		}
		if u.Port < 1 || u.Port > 65535 {
			problems = append(problems, fmt.Sprintf("upstream %q port must be between 1 and 65535", u.Name))
		}
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func parseProbe(raw string) (Probe, error) {
	name, kind, ok := strings.Cut(raw, ":")
	if !ok || strings.TrimSpace(name) == "" {
		return Probe{}, errors.New("expected name:type")
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if !strings.HasSuffix(name, ".") {
		name += "."
	}
	switch strings.ToUpper(strings.TrimSpace(kind)) {
	case "A":
		return Probe{Name: name, QType: 1}, nil
	case "AAAA":
		return Probe{Name: name, QType: 28}, nil
	default:
		return Probe{}, errors.New("type must be A or AAAA")
	}
}

func boolOption(s section, key string, fallback bool) (bool, error) {
	raw, ok := s.options[key]
	if !ok {
		return fallback, nil
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be a boolean", key)
	}
}

func intOption(s section, key string, fallback int) (int, error) {
	raw, ok := s.options[key]
	if !ok {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return value, nil
}

func stringOption(s section, key, fallback string) string {
	if value, ok := s.options[key]; ok {
		return value
	}
	return fallback
}

func stripComment(line string) string {
	var quote rune
	for i, r := range line {
		switch {
		case quote == 0 && (r == '\'' || r == '"'):
			quote = r
		case quote != 0 && r == quote:
			quote = 0
		case quote == 0 && r == '#':
			return line[:i]
		}
	}
	return line
}

func splitUCI(line string) ([]string, error) {
	var fields []string
	var b strings.Builder
	var quote rune
	escaped := false
	flush := func() {
		if b.Len() > 0 {
			fields = append(fields, b.String())
			b.Reset()
		}
	}
	for _, r := range line {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
			}
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case ' ', '\t':
			flush()
		default:
			b.WriteRune(r)
		}
	}
	if escaped || quote != 0 {
		return nil, errors.New("unterminated quote or escape")
	}
	flush()
	if len(fields) == 0 {
		return nil, errors.New("empty directive")
	}
	return fields, nil
}
