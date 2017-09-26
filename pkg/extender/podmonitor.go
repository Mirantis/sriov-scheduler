package extender

import (
	"time"

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
	if !ext.selector(pod) {
		return
	}
	ext.Lock()
	defer ext.Unlock()
	ext.allocatedVFs[pod.Spec.NodeName].Sub(*singleItem)
	ext.purgeByUID(pod.UID)
}

func (ext *Extender) syncAllocated(obj interface{}) {
	pod := obj.(*v1.Pod)
	if !ext.selector(pod) {
		return
	}
	ext.Lock()
	defer ext.Unlock()
	ext.allocatedVFs[pod.Spec.NodeName].Add(*singleItem)
	ext.purgeByUID(pod.UID)
}

func (ext *Extender) syncAllocatedFromUpdated(old, new interface{}) {
	ext.syncAllocated(new)
}
