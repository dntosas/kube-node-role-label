// Package cmd holds the command-line configuration of kube-node-role-label.
package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/client-go/util/homedir"
)

// Build information, injected at link time by goreleaser / the Makefile.
var (
	Version    = "development"
	CommitHash = "unknown"
)

// Log output formats.
const (
	LogFormatJSON = "json"
	LogFormatText = "text"
)

// Config is the parsed command-line configuration.
type Config struct {
	// Labels are the node label keys whose values become node roles.
	Labels []string
	// Interval between reconciliation runs. Zero means run once and exit.
	Interval time.Duration
	// Kubeconfig is the path used when running outside a cluster.
	Kubeconfig string
	// LogLevel is the minimum level that gets emitted.
	LogLevel slog.Level
	// LogFormat is either "json" or "text".
	LogFormat string
	// ShowVersion prints build information and exits.
	ShowVersion bool
}

// ErrHelp is returned when the user asked for -h/--help.
var ErrHelp = flag.ErrHelp

// ParseFlags parses args (without the program name) into a Config.
// Usage and error output is written to w.
func ParseFlags(args []string, w io.Writer) (*Config, error) {
	cfg := &Config{}
	var (
		labels   string
		logLevel string
		verbose  bool
	)

	fs := flag.NewFlagSet("kube-node-role-label", flag.ContinueOnError)
	fs.SetOutput(w)

	fs.StringVar(&labels, "label", "",
		"Comma-separated node label keys to watch. For every node carrying key=value,\n"+
			"the role label node-role.kubernetes.io/<value>=true is added.\n"+
			"Example: -label node-type,karpenter.sh/nodepool")
	fs.DurationVar(&cfg.Interval, "interval", 0,
		"Run as a daemon and reconcile every interval (e.g. 30s, 5m, 1h). Zero runs once and exits.")
	fs.StringVar(&cfg.Kubeconfig, "kubeconfig", defaultKubeconfig(),
		"Path to a kubeconfig file. Only used when in-cluster configuration is unavailable.")
	fs.StringVar(&logLevel, "log-level", "info", "Log level: debug, info, warn or error.")
	fs.StringVar(&cfg.LogFormat, "log-format", LogFormatJSON, "Log format: json or text.")
	fs.BoolVar(&verbose, "v", false, "Shorthand for -log-level debug.")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "Print version information and exit.")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if cfg.ShowVersion {
		return cfg, nil
	}

	cfg.Labels = splitLabels(labels)
	if len(cfg.Labels) == 0 {
		return nil, errors.New("-label is required (run with -h for help)")
	}
	if cfg.Interval < 0 {
		return nil, fmt.Errorf("-interval must not be negative, got %s", cfg.Interval)
	}

	cfg.LogFormat = strings.ToLower(cfg.LogFormat)
	if cfg.LogFormat != LogFormatJSON && cfg.LogFormat != LogFormatText {
		return nil, fmt.Errorf("-log-format must be %q or %q, got %q", LogFormatJSON, LogFormatText, cfg.LogFormat)
	}

	if verbose {
		logLevel = "debug"
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(logLevel)); err != nil {
		return nil, fmt.Errorf("-log-level: %w", err)
	}

	return cfg, nil
}

// NewLogger builds a slog.Logger according to the configuration.
func (c *Config) NewLogger(w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: c.LogLevel}
	var h slog.Handler
	if c.LogFormat == LogFormatText {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(h)
}

// VersionString returns a human readable build description.
func VersionString() string {
	return fmt.Sprintf("kube-node-role-label %s (commit %s)", Version, CommitHash)
}

func splitLabels(s string) []string {
	var out []string
	for _, l := range strings.Split(s, ",") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func defaultKubeconfig() string {
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env
	}
	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	return ""
}
