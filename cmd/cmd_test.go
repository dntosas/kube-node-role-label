package cmd

import (
	"bytes"
	"errors"
	"log/slog"
	"reflect"
	"testing"
	"time"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    *Config
		wantErr bool
	}{
		{
			name: "defaults",
			args: []string{"-label", "node-type"},
			want: &Config{Labels: []string{"node-type"}, LogLevel: slog.LevelInfo, LogFormat: LogFormatJSON},
		},
		{
			name: "multiple labels with spaces and trailing comma",
			args: []string{"-label", "node-type, karpenter.sh/nodepool ,", "-interval", "5m"},
			want: &Config{Labels: []string{"node-type", "karpenter.sh/nodepool"}, Interval: 5 * time.Minute, LogLevel: slog.LevelInfo, LogFormat: LogFormatJSON},
		},
		{
			name: "text format and explicit level",
			args: []string{"-label", "a", "-log-format", "TEXT", "-log-level", "warn"},
			want: &Config{Labels: []string{"a"}, LogLevel: slog.LevelWarn, LogFormat: LogFormatText},
		},
		{
			name: "-v forces debug",
			args: []string{"-label", "a", "-v", "-log-level", "error"},
			want: &Config{Labels: []string{"a"}, LogLevel: slog.LevelDebug, LogFormat: LogFormatJSON},
		},
		{
			name: "version short-circuits validation",
			args: []string{"-version"},
			want: &Config{ShowVersion: true, LogFormat: LogFormatJSON},
		},
		{name: "missing label", args: []string{}, wantErr: true},
		{name: "empty label", args: []string{"-label", " , "}, wantErr: true},
		{name: "negative interval", args: []string{"-label", "a", "-interval", "-1s"}, wantErr: true},
		{name: "bad interval", args: []string{"-label", "a", "-interval", "soon"}, wantErr: true},
		{name: "bad format", args: []string{"-label", "a", "-log-format", "yaml"}, wantErr: true},
		{name: "bad level", args: []string{"-label", "a", "-log-level", "loud"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			got, err := ParseFlags(tt.args, &out)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got.Kubeconfig = "" // environment dependent
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseFlags_Help(t *testing.T) {
	var out bytes.Buffer
	_, err := ParseFlags([]string{"-h"}, &out)
	if !errors.Is(err, ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}
	if out.Len() == 0 {
		t.Error("expected usage to be printed")
	}
}

func TestNewLogger(t *testing.T) {
	var out bytes.Buffer
	cfg := &Config{LogLevel: slog.LevelInfo, LogFormat: LogFormatJSON}
	log := cfg.NewLogger(&out)
	log.Debug("hidden")
	log.Info("shown", "k", "v")
	if got := out.String(); !bytes.Contains([]byte(got), []byte(`"msg":"shown"`)) || bytes.Contains([]byte(got), []byte("hidden")) {
		t.Errorf("unexpected output: %s", got)
	}
}
