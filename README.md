[![CI](https://github.com/dntosas/astrolavos/actions/workflows/go-ci.yml/badge.svg?branch=main)](https://github.com/dntosas/astrolavos/actions/workflows/go-ci.yml) | [![Go Report](https://goreportcard.com/badge/github.com/dntosas/astrolavos)](https://goreportcard.com/badge/github.com/dntosas/astrolavos) | [![Go Release](https://github.com/dntosas/astrolavos/actions/workflows/go-release.yml/badge.svg)](https://github.com/dntosas/astrolavos/actions/workflows/go-release.yml) | [![e2e Tests](https://github.com/dntosas/astrolavos/actions/workflows/e2e.yml/badge.svg)](https://github.com/dntosas/astrolavos/actions/workflows/e2e.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

# kube-node-role-label

This is a proud fork of [label-watch](https://github.com/kolikons/label-watch) repo, with just keeping things up-to-date and multi-arch :heart:

When Kubernetes cluster's created, worker nodes is tagged as none.

`kube-node-role-label` checks a specific label on worker node then create an label `node-role.kubernetes.io/***`

---

## Usage of kube-node-role-label
kube-node-role-label supports two mode of running. The first one is outside kubernetes cluster and inside

#### Example kube-node-role-label outside kuberntes cluster:
1. You must have `kube config` that uses for connecting `kubectl`
2. Run command with the following flags:
```sh
$ kubectl get node
AME                 STATUS     ROLES                  AGE   VERSION
kind-control-plane   Ready      control-plane,master   39s   v1.20.2
kind-worker          NotReady   <none>                 8s    v1.20.2
kind-worker2         NotReady   <none>                 8s    v1.20.2
$ kubectl get node --show label
kind-control-plane   Ready    control-plane,master   54s   v1.20.2   beta.kubernetes.io/arch=amd64,beta.kubernetes.io/os=linux,kubernetes.io/arch=amd64,kubernetes.io/hostname=kind-control-plane,kubernetes.io/os=linux,node-role.kubernetes.io/control-plane=,node-role.kubernetes.io/master=
kind-worker          Ready    <none>                 23s   v1.20.2   beta.kubernetes.io/arch=amd64,beta.kubernetes.io/os=linux,kubernetes.io/arch=amd64,kubernetes.io/hostname=kind-worker,kubernetes.io/os=linux
kind-worker2         Ready    <none>                 23s   v1.20.2   beta.kubernetes.io/arch=amd64,beta.kubernetes.io/os=linux,kubernetes.io/arch=amd64,kubernetes.io/hostname=kind-worker2,kubernetes.io/os=linux
$ kubectl label node kind-worker node-type="worker"
node/kind-worker labeled
$ kubectl label node kind-worker group="infra"
node/kind-worker labeled
$ kubectl label node kind-worker type="backend"
node/kind-worker labeled
$ kube-node-role-label -kubeconfig ~/.kube/config -label node-type,group,type
Running kube-node-role-label
Node kind-worker has been labelled successfully. Label: node-role.kubernetes.io/worker=true
Node kind-worker has been labelled successfully. Label: node-role.kubernetes.io/infra=true
Node kind-worker has been labelled successfully. Label: node-role.kubernetes.io/backend=true
$ kubectl get node
NAME                 STATUS   ROLES                  AGE     VERSION
kind-control-plane   Ready    control-plane,master   3m7s    v1.20.2
kind-worker          Ready    backend,infra,worker   2m36s   v1.20.2
kind-worker2         Ready    <none>                 2m36s   v1.20.2
```

#### Example kube-node-role-label inside kubernetes cluster:
1. Modify ARGs in [scripts/deployment.yml](scripts/deployment.yml#22)
2. Deploy the kubernetes manifest from [scripts/deployment.yml](scripts/deployment.yml)
```sh
$ kubectl get node
NAME                 STATUS   ROLES                  AGE   VERSION
kind-control-plane   Ready    control-plane,master   13m   v1.20.2
kind-worker          Ready    <none>                 12m   v1.20.2
kind-worker2         Ready    <none>                 12m   v1.20.2
$ kubectl apply -f deployment.yml
deployment.apps/kube-node-role-label configured
serviceaccount/kube-node-role-label created
clusterrole.rbac.authorization.k8s.io/kube-node-role-label created
clusterrolebinding.rbac.authorization.k8s.io/kube-node-role-label created
$ kubectl get node
NAME                 STATUS   ROLES                  AGE   VERSION
kind-control-plane   Ready    control-plane,master   14m   v1.20.2
kind-worker          Ready    worker                 13m   v1.20.2
kind-worker2         Ready    worker                 13m   v1.20.2
```

## Installation

**Helm**

```console
$ helm repo add kube-node-role-label https://dntosas.github.io/kube-node-role-label/
$ helm repo update
$ helm upgrade -i kube-node-role-label/kube-node-role-label kube-node-role-label/kube-node-role-label
```

## kube-node-role-label ARGS
```sh
kube-node-role-label --help
Usage of kube-node-role-label:
  -interval string
    	(optional) Start application in deamon mode. Supports format: 's', 'm', 'h'.
  -kubeconfig string
    	(optional) absolute path to the kubeconfig file
  -label string
    	Label that's checking on worker nodes then set label in format node-role.kubernetes.io/VALUE_FROM_LABEL=true.
    	Supports multiple labels: -label node-type,type,etc
    	Example:
    	$ kubectl get node NODE -o jsonpath='{.metadata.labels}' | jq
    	{
    		"beta.kubernetes.io/arch": "amd64",
    		....
    		"node-type": "worker"
    	}
    	$ kube-node-role-label -label node-type
    	$ kubectl get node NODE -o jsonpath='{.metadata.labels}' | jq
    	{
    		"beta.kubernetes.io/arch": "amd64",
    		....
    		"node-type": "worker",
    		"node-role.kubernetes.io/worker": "true"
    	}
  -v	Makes verbose output
```

---
