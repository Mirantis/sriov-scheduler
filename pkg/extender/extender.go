package extender

import (
	"fmt"
	"log"
	"sync"

	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

const (
	TotalVFsResource v1.ResourceName = "totalvfs"
)

var (
	singleItem = resource.NewQuantity(1, resource.DecimalSI)
	zero       = resource.NewQuantity(0, resource.DecimalSI)
)

func NewExtender(client *kubernetes.Clientset) *Extender {
	return &Extender{
		client:       client,
		allocatedVFs: make(map[string]*resource.Quantity),
		promises:     NewPromises(),
		selector:     NetworkSelector,
	}
}

type Extender struct {
	client *kubernetes.Clientset

	sync.Mutex
	allocatedVFs map[string]*resource.Quantity
	promises     PromisesInterface

	selector Selector
}

func (ext *Extender) FilterArgs(args *ExtenderArgs) (*ExtenderFilterResult, error) {
	log.Printf("Filter called with pod %s/%s and args %v", args.Pod.Namespace, args.Pod.Name, args)
	if !ext.selector(&args.Pod) {
		return nil, nil
	}
	ext.Lock()
	defer ext.Unlock()
	result := &ExtenderFilterResult{
		Nodes:       &v1.NodeList{Items: make([]v1.Node, 0, 1)},
		FailedNodes: make(map[string]string),
	}
	for {
		var waitChan chan struct{}
		promised := ext.promises.PromisesCount()
		if promised.Cmp(*zero) == 1 {
			waitChan = make(chan struct{})
			ext.promises.Subscribe(waitChan)
		}
		for _, node := range args.Nodes.Items {
			log.Printf("Checking node %s", node.Name)
			if _, exists := ext.allocatedVFs[node.Name]; !exists {
				ext.allocatedVFs[node.Name] = resource.NewQuantity(0, resource.DecimalSI)
			}
			allocated := ext.allocatedVFs[node.Name]
			if res, exists := node.Status.Allocatable[TotalVFsResource]; !exists {
				log.Printf("No allocatable vfs on a node %s \n", node.Name)
				continue
			} else {
				log.Printf("Node %s has a total of %v allocatable vfs.", node.Name, &res)
				res.Sub(*allocated)
				res.Sub(*promised)
				if res.Cmp(*zero) == 1 {
					log.Printf(
						"Node %s has an available VF and it will be promised to a pod %s/%s.",
						node.Name, args.Pod.Namespace, args.Pod.Name)
					result.Nodes.Items = append(result.Nodes.Items, node)
				} else {
					log.Printf("Node %s doesnt have sufficient number of VFs", node.Name)
					result.FailedNodes[node.Name] = fmt.Sprintf(
						"Not sufficient number of VFs. Allocated: %v. Promised: %v. Total: %v",
						allocated, promised, res,
					)
				}
			}
		}
		if len(result.Nodes.Items) == 0 {
			result.Error = "No nodes have available VFs."
		} else {
			ext.promises.MakePromise(args.Pod.UID)
		}
		if len(result.Error) != 0 && promised.Cmp(*zero) == 1 {
			log.Println("Some VFs are promised to other pods. We will wait until one will be released.")
			if err := WaitFor(waitChan, defaultPromisesCleanerInterval); err != nil {
				return result, nil
			}
			continue
		}
		return result, nil
	}
}

func (ext *Extender) RunPromisesCleaner(interval time.Duration, stopCh <-chan struct{}) {
	ext.promises.RunPromisesCleaner(interval, stopCh)
}

func WaitFor(waitChan <-chan struct{}, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	select {
	case <-waitChan:
		return nil
	case <-timer.C:
		return fmt.Errorf("timeout error")
	}
}
