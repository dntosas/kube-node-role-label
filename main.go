// Command kube-node-role-label derives node-role.kubernetes.io/* labels from
// existing node labels so that `kubectl get nodes` shows meaningful roles.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dntosas/kube-node-role-label/cmd"
	"github.com/dntosas/kube-node-role-label/pkg/label"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cfg, err := cmd.ParseFlags(args, os.Stderr)
	if err != nil {
		if errors.Is(err, cmd.ErrHelp) {
			return 0
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}
	if cfg.ShowVersion {
		fmt.Println(cmd.VersionString())
		return 0
	}

	log := cfg.NewLogger(os.Stdout)
	log.Info("starting", "version", cmd.Version, "commit", cmd.CommitHash, "labels", cfg.Labels)

	client, err := label.NewClient(cfg.Kubeconfig, cmd.VersionString())
	if err != nil {
		log.Error("cannot create kubernetes client", "error", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	labeler := label.New(client, cfg.Labels, log)

	if cfg.Interval == 0 {
		res, err := labeler.Run(ctx)
		log.Info("run finished", "nodes", res.Nodes, "patched", res.Patched, "up_to_date", res.UpToDate, "failed", res.Failed)
		if err != nil {
			log.Error("run finished with errors", "error", err)
			return 1
		}
		return 0
	}

	label.RunLoop(ctx, labeler, cfg.Interval, log)
	return 0
}
