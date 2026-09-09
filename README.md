# kube-node-role-label

[![Go CI](https://github.com/dntosas/kube-node-role-label/actions/workflows/go-ci.yml/badge.svg?branch=main)](https://github.com/dntosas/kube-node-role-label/actions/workflows/go-ci.yml)
[![E2E](https://github.com/dntosas/kube-node-role-label/actions/workflows/e2e.yml/badge.svg?branch=main)](https://github.com/dntosas/kube-node-role-label/actions/workflows/e2e.yml)
[![Release](https://github.com/dntosas/kube-node-role-label/actions/workflows/go-release.yml/badge.svg)](https://github.com/dntosas/kube-node-role-label/actions/workflows/go-release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/dntosas/kube-node-role-label)](https://goreportcard.com/report/github.com/dntosas/kube-node-role-label)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Worker nodes join a Kubernetes cluster with an empty `ROLES` column. Node pools,
however, almost always carry a label that says what the node is for
(`node-type=worker`, `karpenter.sh/nodepool=system`, `eks.amazonaws.com/nodegroup=...`).

`kube-node-role-label` watches one or more of those label keys and, for every
node carrying `key=value`, adds `node-role.kubernetes.io/<value>=true`, so that:

```console
$ kubectl get nodes
NAME                           STATUS   ROLES                AGE   VERSION
ip-10-80-95-36.ec2.internal    Ready    system               3d    v1.33.1
ip-10-80-73-7.ec2.internal     Ready    vector-aggregator    3d    v1.33.1
ip-10-80-12-4.ec2.internal     Ready    control-plane        3d    v1.33.1
```

Control-plane nodes are never touched. Nodes that already carry the right role
are skipped without a patch and without a log line, so a fully labelled cluster
is silent at `info` level.

This started as a fork of [kolikons/label-watch](https://github.com/kolikons/label-watch);
it has since been rewritten around `client-go`, structured logging and a Helm chart.

## Installation

### Helm

```console
helm repo add kube-node-role-label https://dntosas.github.io/kube-node-role-label/
helm repo update
helm upgrade --install kube-node-role-label kube-node-role-label/kube-node-role-label \
  --namespace kube-utils --create-namespace \
  --set label_watch.labels=karpenter.sh/nodepool
```

All chart values are documented in [`charts/kube-node-role-label/README.md`](charts/kube-node-role-label/README.md).

### Container image

Multi-arch (`linux/amd64`, `linux/arm64`) images are published to
`ghcr.io/dntosas/kube-node-role-label:<version>` on every release.

### Binary

Pre-built binaries for Linux and macOS are attached to each
[GitHub release](https://github.com/dntosas/kube-node-role-label/releases), or:

```console
go install github.com/dntosas/kube-node-role-label@latest
```

## Usage

```console
$ kube-node-role-label -h
Usage of kube-node-role-label:
  -interval duration
        Run as a daemon and reconcile every interval (e.g. 30s, 5m, 1h). Zero runs once and exits.
  -kubeconfig string
        Path to a kubeconfig file. Only used when in-cluster configuration is unavailable. (default "$HOME/.kube/config")
  -label string
        Comma-separated node label keys to watch. For every node carrying key=value,
        the role label node-role.kubernetes.io/<value>=true is added.
        Example: -label node-type,karpenter.sh/nodepool
  -log-format string
        Log format: json or text. (default "json")
  -log-level string
        Log level: debug, info, warn or error. (default "info")
  -v    Shorthand for -log-level debug.
  -version
        Print version information and exit.
```

Run once from your workstation against the current kubeconfig context:

```console
$ kubectl label node kind-worker node-type=worker
$ kube-node-role-label -label node-type -log-format text
time=... level=INFO msg=starting version=development commit=unknown labels=[node-type]
time=... level=INFO msg="node role set" node=kind-worker label=node-type role=node-role.kubernetes.io/worker
time=... level=INFO msg="run finished" nodes=2 patched=1 up_to_date=0 failed=0
```

Run as a daemon inside the cluster (this is what the Helm chart does):

```console
kube-node-role-label -label node-type,karpenter.sh/nodepool -interval 5m
```

In daemon mode transient API errors are logged and retried on the next tick;
the process only exits on `SIGINT`/`SIGTERM`.

### Logging

Logs are structured (`log/slog`) and JSON by default so they can be parsed by
any log pipeline without heuristics.

| Level   | When                                                                   |
|---------|------------------------------------------------------------------------|
| `error` | a node patch or the node list failed                                   |
| `warn`  | a label value cannot be used as a role name (invalid label key syntax) |
| `info`  | process start/stop, and each role label that was actually added        |
| `debug` | every node visited, including already-labelled and unlabelled ones     |

### RBAC

The controller needs `list` and `patch` on `nodes` at cluster scope. The chart
creates a `ClusterRole` and `ClusterRoleBinding` for its `ServiceAccount`.

## Development

```console
make help           # list all targets
make ci             # fmt, vet, lint, test
make build          # bin/kube-node-role-label for the host platform
make run LABELS=node-type
make helm-lint      # lint + render the chart
make helm-docs      # regenerate charts/*/README.md from values.yaml
scripts/e2e.sh      # full end-to-end test against a local kind cluster
```

Requires Go (see `go.mod` for the minimum version). `golangci-lint`,
`helm-docs` and `goreleaser` are run through `go run` and need no separate
installation. The e2e script additionally needs `docker`, `kind`, `kubectl`
and `helm`.

## Releasing

1. Merge to `main`. Chart changes under `charts/` are published to the Helm
   repository by `chart-releaser` automatically.
2. Tag the binary/image release: `git tag vX.Y.Z && git push --tags`. GoReleaser
   builds the archives, multi-arch image and GitHub release with a changelog
   grouped by [Conventional Commits](https://www.conventionalcommits.org/) type.

Keep `appVersion` in `charts/kube-node-role-label/Chart.yaml` in sync with the
image tag you release.

## License

[MIT](LICENSE)
