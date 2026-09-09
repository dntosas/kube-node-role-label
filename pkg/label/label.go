// Package label reconciles node-role.kubernetes.io/* labels from arbitrary node labels.
package label

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
)

// RolePrefix is the well-known prefix kubectl uses to render the ROLES column.
const RolePrefix = "node-role.kubernetes.io/"

// RoleValue is the value written to every role label we manage.
const RoleValue = "true"

// controlPlaneSelector excludes control-plane nodes (both the legacy and current role labels).
const controlPlaneSelector = "!" + RolePrefix + "master,!" + RolePrefix + "control-plane"

// Labeler adds node-role labels derived from watched node labels.
type Labeler struct {
	client kubernetes.Interface
	labels []string
	log    *slog.Logger
}

// Result summarises a single reconciliation run.
type Result struct {
	// Nodes is the number of worker nodes inspected.
	Nodes int
	// Patched is the number of role labels that were added or corrected.
	Patched int
	// UpToDate is the number of role labels that were already correct.
	UpToDate int
	// Failed is the number of patches that returned an error.
	Failed int
}

// New returns a Labeler that watches the given label keys.
func New(client kubernetes.Interface, labels []string, log *slog.Logger) *Labeler {
	if log == nil {
		log = slog.Default()
	}
	return &Labeler{client: client, labels: labels, log: log}
}

// Run performs one reconciliation pass over all worker nodes.
//
// Nodes that already carry the desired role label are left untouched and
// only reported at debug level, so a steady-state cluster produces no
// info-level output. Patch failures are logged, counted and returned as a
// joined error after every node has been visited.
func (l *Labeler) Run(ctx context.Context) (Result, error) {
	var res Result

	nodes, err := l.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: controlPlaneSelector})
	if err != nil {
		return res, fmt.Errorf("list nodes: %w", err)
	}
	res.Nodes = len(nodes.Items)

	var errs []error
	for i := range nodes.Items {
		node := &nodes.Items[i]
		for _, key := range l.labels {
			if err := l.reconcile(ctx, node, key, &res); err != nil {
				errs = append(errs, err)
			}
		}
	}

	l.log.Debug("reconciliation finished",
		"nodes", res.Nodes, "patched", res.Patched, "up_to_date", res.UpToDate, "failed", res.Failed)

	return res, errors.Join(errs...)
}

func (l *Labeler) reconcile(ctx context.Context, node *corev1.Node, key string, res *Result) error {
	log := l.log.With("node", node.Name, "label", key)

	role, ok := node.Labels[key]
	if !ok {
		log.Debug("watched label not present on node")
		return nil
	}

	roleKey := RolePrefix + role
	if msgs := validation.IsQualifiedName(roleKey); len(msgs) > 0 {
		log.Warn("label value is not a valid node role, skipping", "value", role, "reason", msgs[0])
		return nil
	}

	if node.Labels[roleKey] == RoleValue {
		res.UpToDate++
		log.Debug("node role already set", "role", roleKey)
		return nil
	}

	if err := l.patch(ctx, node.Name, roleKey); err != nil {
		res.Failed++
		log.Error("failed to set node role", "role", roleKey, "error", err)
		return fmt.Errorf("node %s: set %s: %w", node.Name, roleKey, err)
	}

	res.Patched++
	log.Info("node role set", "role", roleKey)
	return nil
}

// patch uses a strategic merge patch so it works even when the node has no labels map yet.
func (l *Labeler) patch(ctx context.Context, node, roleKey string) error {
	body, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"labels": map[string]string{roleKey: RoleValue},
		},
	})
	if err != nil {
		return err
	}
	_, err = l.client.CoreV1().Nodes().Patch(ctx, node, types.StrategicMergePatchType, body, metav1.PatchOptions{
		FieldManager: "kube-node-role-label",
	})
	return err
}
