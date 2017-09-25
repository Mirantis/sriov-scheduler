package extender

import (
	"strings"

	"k8s.io/client-go/pkg/api/v1"
)

type Selector func(pod *v1.Pod) bool

// NetworkSelector decides if pod requires virtual function.
func NetworkSelector(pod *v1.Pod) bool {
	if networksString, exists := pod.Annotations["networks"]; exists {
		networks := strings.Split(networksString, ",")
		for _, net := range networks {
			if net == "sriov" {
				return true
			}
		}
	}
	return false
}
