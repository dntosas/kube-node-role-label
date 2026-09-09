package label

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func node(name string, labels map[string]string) *corev1.Node {
	return &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels}}
}

func nodeLabels(t *testing.T, c *fake.Clientset, name string) map[string]string {
	t.Helper()
	n, err := c.CoreV1().Nodes().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get node %s: %v", name, err)
	}
	return n.Labels
}

func newLogger(level slog.Level) (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: level})), buf
}

func TestRun_AddsRoleLabel(t *testing.T) {
	c := fake.NewClientset(
		node("worker-1", map[string]string{"node-type": "worker"}),
		node("worker-2", map[string]string{"node-type": "infra", "team": "platform"}),
	)
	log, _ := newLogger(slog.LevelDebug)

	res, err := New(c, []string{"node-type", "team"}, log).Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Nodes != 2 || res.Patched != 3 || res.UpToDate != 0 || res.Failed != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}

	if got := nodeLabels(t, c, "worker-1")[RolePrefix+"worker"]; got != RoleValue {
		t.Errorf("worker-1 role label = %q, want %q", got, RoleValue)
	}
	l2 := nodeLabels(t, c, "worker-2")
	if l2[RolePrefix+"infra"] != RoleValue || l2[RolePrefix+"platform"] != RoleValue {
		t.Errorf("worker-2 labels = %v, want infra and platform roles", l2)
	}
}

func TestRun_AlreadyLabelledIsSilentAndNotPatched(t *testing.T) {
	c := fake.NewClientset(
		node("worker-1", map[string]string{"node-type": "worker", RolePrefix + "worker": RoleValue}),
	)
	patches := 0
	c.PrependReactor("patch", "nodes", func(k8stesting.Action) (bool, runtime.Object, error) {
		patches++
		return false, nil, nil
	})
	log, buf := newLogger(slog.LevelInfo)

	res, err := New(c, []string{"node-type"}, log).Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if patches != 0 {
		t.Errorf("expected no patch calls, got %d", patches)
	}
	if res.UpToDate != 1 || res.Patched != 0 {
		t.Errorf("unexpected result: %+v", res)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no info-level output for an up-to-date node, got:\n%s", buf.String())
	}
}

func TestRun_WrongRoleValueIsCorrected(t *testing.T) {
	c := fake.NewClientset(
		node("worker-1", map[string]string{"node-type": "worker", RolePrefix + "worker": "false"}),
	)
	log, _ := newLogger(slog.LevelDebug)

	res, err := New(c, []string{"node-type"}, log).Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Patched != 1 {
		t.Fatalf("expected 1 patch, got %+v", res)
	}
	if got := nodeLabels(t, c, "worker-1")[RolePrefix+"worker"]; got != RoleValue {
		t.Errorf("role label = %q, want %q", got, RoleValue)
	}
}

func TestRun_MissingWatchedLabelIsIgnored(t *testing.T) {
	c := fake.NewClientset(node("worker-1", map[string]string{"other": "x"}))
	log, buf := newLogger(slog.LevelInfo)

	res, err := New(c, []string{"node-type"}, log).Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Patched != 0 || res.UpToDate != 0 {
		t.Errorf("unexpected result: %+v", res)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no info-level output, got:\n%s", buf.String())
	}
	if _, ok := nodeLabels(t, c, "worker-1")[RolePrefix+"x"]; ok {
		t.Error("unexpected role label derived from unrelated label")
	}
}

func TestRun_SkipsControlPlaneNodes(t *testing.T) {
	c := fake.NewClientset(
		node("cp-1", map[string]string{"node-type": "cp", RolePrefix + "control-plane": ""}),
		node("cp-legacy", map[string]string{"node-type": "cp", RolePrefix + "master": ""}),
		node("worker-1", map[string]string{"node-type": "worker"}),
	)
	log, _ := newLogger(slog.LevelDebug)

	res, err := New(c, []string{"node-type"}, log).Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Nodes != 1 || res.Patched != 1 {
		t.Fatalf("expected only the worker to be processed, got %+v", res)
	}
	for _, cp := range []string{"cp-1", "cp-legacy"} {
		if _, ok := nodeLabels(t, c, cp)[RolePrefix+"cp"]; ok {
			t.Errorf("%s: control-plane node must not be patched", cp)
		}
	}
}

func TestRun_InvalidRoleValueIsWarnedAndSkipped(t *testing.T) {
	c := fake.NewClientset(
		node("worker-1", map[string]string{"node-type": "not a valid/label value!"}),
	)
	log, buf := newLogger(slog.LevelInfo)

	res, err := New(c, []string{"node-type"}, log).Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Patched != 0 || res.Failed != 0 {
		t.Errorf("unexpected result: %+v", res)
	}
	if !strings.Contains(buf.String(), "level=WARN") {
		t.Errorf("expected a warning, got:\n%s", buf.String())
	}
}

func TestRun_PatchErrorIsReportedAndOtherNodesContinue(t *testing.T) {
	c := fake.NewClientset(
		node("bad", map[string]string{"node-type": "worker"}),
		node("good", map[string]string{"node-type": "worker"}),
	)
	boom := errors.New("boom")
	c.PrependReactor("patch", "nodes", func(a k8stesting.Action) (bool, runtime.Object, error) {
		if a.(k8stesting.PatchAction).GetName() == "bad" {
			return true, nil, boom
		}
		return false, nil, nil
	})
	log, buf := newLogger(slog.LevelInfo)

	res, err := New(c, []string{"node-type"}, log).Run(context.Background())
	if !errors.Is(err, boom) {
		t.Fatalf("expected wrapped boom error, got %v", err)
	}
	if res.Failed != 1 || res.Patched != 1 {
		t.Errorf("unexpected result: %+v", res)
	}
	if got := nodeLabels(t, c, "good")[RolePrefix+"worker"]; got != RoleValue {
		t.Errorf("good node should still have been patched, labels: %v", nodeLabels(t, c, "good"))
	}
	if !strings.Contains(buf.String(), "level=ERROR") {
		t.Errorf("expected an error log line, got:\n%s", buf.String())
	}
}

func TestRun_ListErrorIsReturned(t *testing.T) {
	c := fake.NewClientset()
	boom := errors.New("api down")
	c.PrependReactor("list", "nodes", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, boom
	})
	log, _ := newLogger(slog.LevelInfo)

	if _, err := New(c, []string{"node-type"}, log).Run(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("expected list error, got %v", err)
	}
}
