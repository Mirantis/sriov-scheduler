Sriov scheduler extension
=========================

## Demo

[![asciicast](https://asciinema.org/a/143079.png)](https://asciinema.org/a/143079)

## Get started

This application does 2 things:
- discovers total number of sriov vfs on every node in the cluster
- prevents scheduler from binding pods on a nodes that doesn't have vfs left

In order to deploy applicaton you need to follow next instructions.
Run a disvery tool on every node in the cluster:
```
kubectl create -f tools/discovery.yaml
``` 

It will deploy daemonset with golang script that will read totalvfs number
on each node and report them to kubernetes. Reported information will be available on a nodes:
```
  status:
    allocatable:
      cpu: "8"
      memory: 32766568Ki
      pods: "110"
      totalvfs: "1"
```

Next deploy scheduler extension itself:
```
kubectl create -f tools/extender.yaml
```
It will create deployment with http server and a service for it.

Important to note that all pods without requested sriov network will be ignored,
in future it will be easy to add any other selection algorithm:

```
kind: Pod
metadata:
  annotations:
    networks: sriov
```

And as a last step we need to change kubernetes scheduler configuration.
On my environment kubernetes scheduler is self-hosted and I will be using
configmap as a policy configuration source.

Create configmap "scheduler-policy" in a kube-system namespace:
```
kubectl create configmap scheduler-policy -n kube-system --from-file=policy.cfg=tools/scheduler.json
```

And add policy-configmap option to kubernetes scheduler:

```
spec:
  containers:
  - command:
    - /hyperkube
    - scheduler
    - --address=127.0.0.1
    - --leader-elect=true
    - --kubeconfig=/etc/kubernetes/scheduler.conf
    - --policy-configmap
    - scheduler-policy
```