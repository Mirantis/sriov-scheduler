package extender

import (
	"log"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

// CreateMonitor creates pod informer.
// This pod informer listens to pod changes and once node name is assigned by scheduler,
// it removes global promise and adds allocated vf to a proper node
func (ext *Extender) CreateMonitor() cache.Controller {
	lw := cache.NewListWatchFromClient(
		ext.client.Core().RESTClient(), "pods", meta_v1.NamespaceAll,
		fields.ParseSelectorOrDie("spec.nodeName!="+""),
	)
	return ext.createMonitorFromSource(lw)
}

func (ext *Extender) createMonitorFromSource(lw cache.ListerWatcher) cache.Controller {
	_, controller := cache.NewInformer(
		lw, &v1.Pod{}, 30*time.Second, cache.ResourceEventHandlerFuncs{
			AddFunc:    ext.syncAllocated,
			UpdateFunc: ext.syncAllocatedFromUpdated,
			DeleteFunc: ext.syncPurged,
		},
	)
	return controller
}

func (ext *Extender) syncPurged(obj interface{}) {
	pod := obj.(*v1.Pod)
	log.Printf("removing pod %s\n", pod.UID)
	if !ext.selector(pod) {
		log.Printf("pod %s skipped\n", pod.UID)
		return
	}
	ext.Lock()
	defer ext.Unlock()
	if _, exists := ext.allocatedVFs[pod.Spec.NodeName]; !exists {
		ext.allocatedVFs[pod.Spec.NodeName] = resource.NewQuantity(0, resource.DecimalSI)
	}
	ext.allocatedVFs[pod.Spec.NodeName].Sub(*singleItem)
	ext.promises.PurgePromise(pod.UID)
	log.Printf(
		"pod %s removed, total vfs for a node %s - %v\n",
		pod.UID, pod.Spec.NodeName, ext.allocatedVFs[pod.Spec.NodeName])
}

func (ext *Extender) syncAllocated(obj interface{}) {
	pod := obj.(*v1.Pod)
	log.Printf("updating pod %s\n", pod.UID)
	if !ext.selector(pod) {
		log.Printf("pod %s skipped\n", pod.UID)
		return
	}
	ext.Lock()
	defer ext.Unlock()
	if _, exists := ext.allocatedVFs[pod.Spec.NodeName]; !exists {
		ext.allocatedVFs[pod.Spec.NodeName] = resource.NewQuantity(0, resource.DecimalSI)
	}
	ext.allocatedVFs[pod.Spec.NodeName].Add(*singleItem)
	ext.promises.PurgePromise(pod.UID)
	log.Printf("pod %s updated\n", pod.UID)
}

func (ext *Extender) syncAllocatedFromUpdated(old, new interface{}) {
	// sync old pod only if it was updated
	if !ext.selector(old.(*v1.Pod)) {
		ext.syncAllocated(new)
	}
}
