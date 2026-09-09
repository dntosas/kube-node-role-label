package label

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestRunLoop_RunsImmediatelyAndStopsOnCancel(t *testing.T) {
	c := fake.NewClientset(node("worker-1", map[string]string{"node-type": "worker"}))
	var lists atomic.Int32
	c.PrependReactor("list", "nodes", func(k8stesting.Action) (bool, runtime.Object, error) {
		lists.Add(1)
		return false, nil, nil
	})
	log := slog.New(slog.DiscardHandler)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		RunLoop(ctx, New(c, []string{"node-type"}, log), 10*time.Millisecond, log)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for lists.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("expected at least 3 reconciliations, got %d", lists.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunLoop did not return after context cancellation")
	}
}
