package label

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dntosas/kube-node-role-label/cmd"
	"golang.org/x/sync/errgroup"
)

// convertTime coverting cmd flags paramst to interval
func convertTime(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		fmt.Printf("Error to convert: %s", err)
		panic(err)
	}
	return d
}

// RunTimerLabel is staring RunLabel() with timer
func RunTimerLabel(c *cmd.Command) {

	ctx, done := context.WithCancel(context.Background())
	g, gctx := errgroup.WithContext(ctx)
	d := convertTime(c.Interval)

	// goroutine to check for signals to gracefully finish all functions
	g.Go(func() error {
		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

		select {
		case sig := <-signalChannel:
			fmt.Printf("Received signal: %s\n", sig)
			done()
		case <-gctx.Done():
			fmt.Printf("Closing signal goroutine\n")
			return gctx.Err()
		}

		return nil
	})

	// just a ticker every interval
	g.Go(func() error {
		ticker := time.NewTicker(d)
		for {
			fmt.Printf("Running label updator\n")
			// Running update label in k8s
			RunLabel(c)

			select {
			case <-ticker.C:
				continue
			case <-gctx.Done():
				fmt.Printf("Closing label updator.\n")
				return gctx.Err()
			}
		}
	})

	// wait for all errgroup goroutines
	err := g.Wait()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Print("kube-node-role-label was canceled")
		} else {
			fmt.Printf("Received error: %v", err)
		}
	} else {
		fmt.Println("kube-node-role-label is done")
	}

}
