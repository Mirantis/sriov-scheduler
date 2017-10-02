package tests

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dshulyak/sriov-scheduler/pkg/extender"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	vfsDevice       = "eth3"
	policyConfigMap = "scheduler-policy"
	sriovTotalVFs   = "2"
)

var (
	kubeconfig          string
	deploymentDirectory string
)

func init() {
	pflag.StringVar(&kubeconfig, "kubeconfig", "", "Kubernetes config")
	pflag.StringVarP(&deploymentDirectory, "deployments", "d", "", "Directory with all deployment definitions")
	pflag.Parse()
}

// TestSriovExtender runs next scenario:
// 1. Read discovery and extender definitions from this repository tools directory
// 2. Patch discovery daemonset with fake sriov_totalvfs mount
// 3. Deploy discovery daemonset and wait until totalvfs resource will be saved on nodes
// 4. Deployment extender deployment and service. Wait until extender pods are ready.
// 5. Update policy config for scheduler.
// 6. Create 5 pods that require sriov network.
// 7. Verify that 4 pods will be running and in ready state.
// 8. And 1 pod won't be scheduled.
func TestSriovExtender(t *testing.T) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(t, err)
	client, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)

	discovery := v1beta1.DaemonSet{}
	discoveryData, err := ioutil.ReadFile(filepath.Join(deploymentDirectory, "discovery.yaml"))
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(discoveryData, &discovery))

	extend := &apps.Deployment{}
	extenderSvc := v1.Service{}
	extenderDataMulti, err := ioutil.ReadFile(filepath.Join(deploymentDirectory, "extender.yaml"))
	require.NoError(t, err)
	extenderData := strings.Split(string(extenderDataMulti), "---\n")
	require.Len(t, extenderData, 2)
	require.NoError(t, yaml.Unmarshal([]byte(extenderData[0]), &extenderSvc))
	require.NoError(t, yaml.Unmarshal([]byte(extenderData[1]), extend))

	sriovTotalVFsQuantity, err := resource.ParseQuantity(sriovTotalVFs)
	require.NoError(t, err)
	fakeVFs := v1.ConfigMap{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "faketotalvfs",
			Namespace: discovery.Namespace,
		},
		Data: map[string]string{"sriov_totalvfs": sriovTotalVFs},
	}
	_, err = client.ConfigMaps(fakeVFs.Namespace).Create(&fakeVFs)
	require.NoError(t, err)

	require.Len(t, discovery.Spec.Template.Spec.Volumes, 1)
	discovery.Spec.Template.Spec.Volumes[0] = v1.Volume{
		Name: "sys",
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: v1.LocalObjectReference{Name: fakeVFs.Name}},
		},
	}
	require.Len(t, discovery.Spec.Template.Spec.Containers, 1)
	require.Len(t, discovery.Spec.Template.Spec.Containers[0].VolumeMounts, 1)
	discovery.Spec.Template.Spec.Containers[0].VolumeMounts[0] = v1.VolumeMount{
		Name:      "sys",
		MountPath: fmt.Sprintf("/sys/class/net/%s/device/", vfsDevice),
	}
	discovery.Spec.Template.Spec.Containers[0].Command = append(
		discovery.Spec.Template.Spec.Containers[0].Command, "-i", vfsDevice)
	_, err = client.DaemonSets(discovery.Namespace).Create(&discovery)
	require.NoError(t, err)
	require.NoError(t, Eventually(func() error {
		nodes, err := client.Nodes().List(meta_v1.ListOptions{})
		if err != nil {
			return err
		}
		for _, node := range nodes.Items {
			if val, exists := node.Status.Allocatable[extender.TotalVFsResource]; !exists {
				return fmt.Errorf("node %s doesnt have totalvfs discovered", node.Name)
			} else {
				if val.Cmp(sriovTotalVFsQuantity) != 0 {
					return fmt.Errorf(
						"discovered quantity %v is different from expected %v on node %s",
						&val, &sriovTotalVFsQuantity, node.Name,
					)
				}
			}
			return nil

		}
	}, 10*time.Second, 500*time.Millisecond))

	extend, err = client.AppsV1beta1().Deployments(extend.Namespace).Create(extender)
	require.NoError(t, err)
	_, err = client.Services(extenderSvc.Namespace).Create(&extenderSvc)
	require.NoError(t, err)
	require.NoError(t, Eventually(func() error {
		pods, err := client.Core().Pods(extend.Namespace).List(meta_v1.ListOptions{
			LabelSelector: extend.Spec.Selector,
		})
		if err != nil {
			return err
		}
		if lth := int32(len(pods.Items)); lth != &extend.Spec.Replicas {
			return fmt.Errorf("unexpected number of replices %d != %d for extender", lth, extend.Spec.Replicas)
		}
		for _, pod := range pods.Items {
			if pod.Status.Phase != v1.PodRunning {
				return fmt.Errorf("pod %v is not yet running", &pod)
			}
		}
		return nil
	}, 10*time.Second, 500*time.Millisecond))

	schedulerPod, err := client.Core().Pods("kube-system").Get("kube-scheduler-kube-master", meta_v1.GetOptions{})
	require.NoError(t, client.Core().Pods("kube-system").Delete(schedulerPod.Name, &meta_v1.DeleteOptions{}))
	require.NoError(t, err)
	require.Len(t, schedulerPod.Spec.Containers, 1)
	schedulerPod.Spec.Containers[0].Command = append(
		schedulerPod.Spec.Containers[0].Command, "--policy-configmap", policyConfigMap)

	policyData, err := ioutil.ReadFile(filepath.Join(deploymentDirectory, "scheduler.yaml"))
	require.NoError(t, err)

	policyCfg := v1.ConfigMap{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      policyConfigMap,
			Namespace: schedulerPod.Namespace,
		},
		Data: map[string]string{"policy.cfg": string(policyData)},
	}
	_, err = client.ConfigMaps(policyCfg.Namespace).Create(&policyCfg)
	require.NoError(t, err)
	schedulerPod, err = client.Core().Pods(schedulerPod.Namespace).Create(schedulerPod)
	require.NoError(t, err)
	require.NoError(t, Eventually(func() error {
		pod, err := client.Core().Pods(schedulerPod.Namespace).Get(schedulerPod.Name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}
		if pod.Status.Phase != v1.PodRunning {
			return fmt.Errorf("scheduler is not running %s", pod)
		}
		return nil
	}, 3*time.Second, 500*time.Millisecond))

	var sriovPods int32 = 5
	sriovDeployment := &apps.Deployment{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:        "sriov-test-deployment",
			Namespace:   "default",
			Annotations: map[string]string{"networks": "sriov"},
		},
		Spec: apps.DeploymentSpec{
			Replicas: &sriovPods,
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "test-pause-container",
							Image: "gcr.io/google_containers/pause:3.0",
						},
					},
				},
			},
		},
	}
	sriovDeployment, err = client.AppsV1beta1().Deployments(sriovDeployment.Namespace).Create(sriovDeployment)
	require.NoError(t, err)
	require.NoError(t, Eventually(func() error {
		pods, err := client.Core().Pods(sriovDeployment.Namespace).List(v1.ListOptions{
			Selector: sriovDeployment.Spec.Selector,
		})
		if err != nil {
			return err
		}
		if int32(len(pods.Items)) != &sriovDeployment.Spec.Replicas {
			return fmt.Errorf("some pods were not yet created for deployment %s", sriovDeployment.Name)
		}
		var running int
		var pending int
		for _, pod := range pods.Items {
			switch pod.Status.Phase {
			case v1.PodRunning:
				running++
			case v1.PodPending:
				pending++
			}
		}
		if running != 4 {
			return fmt.Errorf("unexpected number of running pods %d - %v", running, pods)
		}
		if pending != 1 {
			return fmt.Errorf("unexpected number of pending pods %d - %v", pending, pods)
		}
		return nil
	}, 10*time.Second, 500*time.Millisecond))
}

func Eventually(f func() error, timeout, timeinterval time.Duration) error {
	ticker := time.NewTicker(timeinterval).C
	timer := time.NewTimer(timeout).C
	var err error
	for {
		select {
		case <-ticker:
			err = f()
			if err != nil {
				continue
			}
			return nil
		case <-timer:
			return err
		}
	}
}
