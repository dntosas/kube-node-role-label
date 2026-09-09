package label

import (
	"context"
	"log/slog"
	"time"
)

// RunLoop runs the Labeler immediately and then every interval until ctx is
// cancelled. Transient errors are logged and retried on the next tick; the
// daemon never exits because of a failed reconciliation.
func RunLoop(ctx context.Context, l *Labeler, interval time.Duration, log *slog.Logger) {
	if log == nil {
		log = slog.Default()
	}
	log.Info("starting reconciliation loop", "interval", interval.String(), "labels", l.labels)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if _, err := l.Run(ctx); err != nil && ctx.Err() == nil {
			log.Error("reconciliation failed, will retry", "error", err, "retry_in", interval.String())
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			log.Info("shutting down", "reason", context.Cause(ctx))
			return
		}
	}
}
