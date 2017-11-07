#!/bin/bash

set -o errexit
set -o xtrace

$HOME/.kubeadm-dind-cluster/kubectl get clusterroles system:kube-scheduler -o yaml > /tmp/role.yml
cat tools/configmap-extension.yml >> /tmp/role.yml
$HOME/.kubeadm-dind-cluster/kubectl replace -f /tmp/role.yml
rm /tmp/role.yml
