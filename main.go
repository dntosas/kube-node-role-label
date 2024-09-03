package main

import (
	"fmt"
	"os"

	"github.com/dntosas/kube-node-role-label/cmd"
	"github.com/dntosas/kube-node-role-label/pkg/label"
)

func main() {
	opts, e := cmd.ParseFlags()
	if e != nil {
		fmt.Print(e)
		os.Exit(0)
	}
	fmt.Println("Running kube-node-role-label")
	// Run Lable pkg to set labels
	if opts.Interval == "" {
		label.RunLabel(opts)
	}

	// Run Ticker daemon
	if opts.Interval != "" {
		label.RunTimerLabel(opts)
	}

	os.Exit(0)
}
