package extender

import (
	"log"
	"testing"

	"time"

	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/pkg/api/v1"
	fake "k8s.io/client-go/tools/cache/testing"
)

func TestPodMonitorFunctions(t *testing.T) {
	single := *resource.NewQuantity(1, resource.DecimalSI)
	double := *resource.NewQuantity(2, resource.DecimalSI)

	ext := NewExtender(nil)
	source := fake.NewFakeControllerSource()
	ctl := ext.createMonitorFromSource(source)
	stopCh := make(chan struct{})
	defer func() {
		close(stopCh)
	}()
	log.Println("running controller")
	go ctl.Run(stopCh)
	podWithSriov := &v1.Pod{ObjectMeta: metav1.ObjectMeta{
		UID:         types.UID("1"),
		Name:        "1",
		Annotations: map[string]string{"networks": "sriov"}},
		Spec: v1.PodSpec{NodeName: "node1"},
	}
	podWithoutSriov := &v1.Pod{ObjectMeta: metav1.ObjectMeta{
		UID:         types.UID("2"),
		Name:        "2",
		Annotations: map[string]string{"networks": "calico"}},
		Spec: v1.PodSpec{NodeName: "node1"}}
	source.Add(podWithSriov)
	source.Add(podWithoutSriov)
	log.Println("verifying that only single vf will be allocated")
	Eventually(t, func() error {
		ext.Lock()
		defer ext.Unlock()
		if ext.allocatedVFs["node1"].Cmp(single) != 0 {
			return fmt.Errorf("Expected one allocated VFs on node1")
		}
		return nil
	}, 10*time.Millisecond, 2*time.Millisecond)
	podWithoutSriov.Annotations["networks"] = "calico,sriov"
	source.Modify(podWithoutSriov)
	log.Println("verifying that after update 2 vfs will be allocated")
	Eventually(t, func() error {
		ext.Lock()
		defer ext.Unlock()
		if ext.allocatedVFs["node1"].Cmp(double) != 0 {
			return fmt.Errorf("Expected two allocated VFs on node1")
		}
		return nil
	}, 10*time.Millisecond, 2*time.Millisecond)
	source.Delete(podWithSriov)
	source.Delete(podWithoutSriov)
	log.Println("verifying that after deletion allocated vfs will be cleaned")
	Eventually(t, func() error {
		ext.Lock()
		defer ext.Unlock()
		if !ext.allocatedVFs["node1"].IsZero() {
			return fmt.Errorf("Expected no allocated VFs on node1, got %v", ext.allocatedVFs["node1"])
		}
		return nil
	}, 10*time.Millisecond, 2*time.Millisecond)
}

func Eventually(t *testing.T, f func() error, timeout, interval time.Duration) {
	ticker := time.NewTicker(interval).C
	timer := time.NewTimer(timeout).C
	var err error
	for {
		select {
		case <-ticker:
			err = f()
			if err == nil {
				return
			}
			log.Println(err.Error())
		case <-timer:
			t.Errorf("Timeout error: %v\n", err)
			return
		}
	}
}
