#!/bin/bash

kubectl delete configmap faketotalvfs -n kube-system
kubectl delete daemonset sriov-discovery -n kube-system
kubectl delete deployment sriov-scheduler-extender -n kube-system
kubectl delete service sriov-scheduler-extender -n kube-system
kubectl delete configmap scheduler-policy -n kube-system
kubectl delete pod sriov-test-kube-scheduler-kube-master -n kube-system
kubectl delete deployment sriov-test-deployment
docker exec kube-master mv /tmp/kube-scheduler.yaml /etc/kubernetes/manifests
