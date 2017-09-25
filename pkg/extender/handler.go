package extender

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/pkg/api/v1"
)

const (
	TotalVFsResource v1.ResourceName = "totalvfs"
)

var singleItem resource.Quantity

func init() {
	singleItem, _ = resource.ParseQuantity("1")
}

func MakeServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/filter", NewExtender())
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	return srv
}

func NewExtender() *Extender {
	return &Extender{}
}

type Extender struct {
	sync.Mutex
	allocatedVFs map[string]*resource.Quantity

	// number of promises must be always equal to number of promised VFs
	// in separate loop we will go over promises and clear them as needed
	promises    []time.Time
	promisedVFs map[string]*resource.Quantity

	selector Selector
}

func (ext *Extender) FilterArgs(args *ExtenderArgs) (*ExtenderFilterResult, error) {
	if !ext.selector(&args.Pod) {
		return nil, nil
	}
	ext.Lock()
	defer ext.Unlock()
	result := &ExtenderFilterResult{
		Nodes:       &v1.NodeList{Items: make([]v1.Node, 0, 1)},
		FailedNodes: make(map[string]string),
	}
	for _, node := range args.Nodes.Items {
		if _, exists := ext.allocatedVFs[node.Name]; !exists {
			ext.allocatedVFs[node.Name] = resource.NewQuantity(0, resource.DecimalExponent)
			ext.promisedVFs[node.Name] = resource.NewQuantity(0, resource.DecimalExponent)
		}
		allocated := ext.allocatedVFs[node.Name]
		promised := ext.promisedVFs[node.Name]
		if res, exists := node.Status.Allocatable[TotalVFsResource]; !exists {
			log.Printf("No allocatable vfs on a node %s \n", node.Name)
			continue
		} else {
			totalVFs, _ := res.AsInt64()
			log.Printf("Node %s has a total of %d allocatable vfs.", node.Name, totalVFs)
			res.Sub(*allocated)
			res.Sub(*promised)
			restVFs, _ := res.AsInt64()
			if restVFs > int64(0) {
				log.Printf(
					"Node %s has available VF and it will be promised to a pod %s/%s.",
					node.Name, args.Pod.Namespace, args.Pod.Name)
				result.Nodes.Items = append(result.Nodes.Items, node)
				promised.Sub(singleItem)
				ext.promises = append(ext.promises, time.Now())
			} else {
				log.Printf("Node %s doesnt have sufficient number of VFs", node.Name)
				result.FailedNodes[node.Name] = fmt.Sprintf(
					"Not sufficient number of VFs. Allocated: %v. Promised: %v. Total: %v",
					allocated, promised, totalVFs,
				)
			}
		}
	}
	if len(result.Nodes.Items) == 0 {
		result.Error = "No nodes have available VFs."
	}
	return result, nil
}

func (ext *Extender) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var args ExtenderArgs
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	if err := json.Unmarshal(body, &args); err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	if result, err := ext.FilterArgs(&args); err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	} else {
		body, err := json.Marshal(result)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(200)
		w.Write(body)
	}
}

// ExtenderArgs represents the arguments needed by the extender to filter/prioritize
// nodes for a pod.
type ExtenderArgs struct {
	// Pod being scheduled
	Pod v1.Pod
	// List of candidate nodes where the pod can be scheduled; to be populated
	// only if ExtenderConfig.NodeCacheCapable == false
	Nodes *v1.NodeList
	// List of candidate node names where the pod can be scheduled; to be
	// populated only if ExtenderConfig.NodeCacheCapable == true
	NodeNames *[]string
}

// FailedNodesMap represents the filtered out nodes, with node names and failure messages
type FailedNodesMap map[string]string

// ExtenderFilterResult represents the results of a filter call to an extender
type ExtenderFilterResult struct {
	// Filtered set of nodes where the pod can be scheduled; to be populated
	// only if ExtenderConfig.NodeCacheCapable == false
	Nodes *v1.NodeList
	// Filtered set of nodes where the pod can be scheduled; to be populated
	// only if ExtenderConfig.NodeCacheCapable == true
	NodeNames *[]string
	// Filtered out nodes where the pod can't be scheduled and the failure messages
	FailedNodes FailedNodesMap
	// Error message indicating failure
	Error string
}
