package extender

import (
	"strconv"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/pkg/api/v1"
)

func TestFilter(t *testing.T) {
	testCases := []struct {
		nodesResources  []int64
		alreadyPromised *resource.Quantity
		failedNodes     []string
		error           bool
	}{
		{
			nodesResources:  []int64{1, 1, 0},
			alreadyPromised: resource.NewQuantity(1, resource.DecimalSI),
			failedNodes:     []string{"0", "1", "2"},
			error:           true,
		},
		{
			nodesResources:  []int64{2, 0},
			alreadyPromised: resource.NewQuantity(1, resource.DecimalSI),
			failedNodes:     []string{"1"},
		},
		{
			nodesResources:  []int64{3, 3},
			alreadyPromised: resource.NewQuantity(0, resource.DecimalSI),
		},
	}
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ext := NewExtender(nil)
			ext.promisedVFs = tc.alreadyPromised
			args := ExtenderArgs{
				Pod: v1.Pod{
					ObjectMeta: meta_v1.ObjectMeta{
						UID:         types.UID("1"),
						Annotations: map[string]string{"networks": "sriov"},
					}},
				Nodes: &v1.NodeList{Items: []v1.Node{}},
			}
			for i, totalvfs := range tc.nodesResources {
				args.Nodes.Items = append(args.Nodes.Items, v1.Node{
					ObjectMeta: meta_v1.ObjectMeta{Name: strconv.Itoa(i)},
					Status: v1.NodeStatus{
						Allocatable: v1.ResourceList{
							TotalVFsResource: *resource.NewQuantity(
								totalvfs, resource.DecimalSI)},
					},
				})
			}
			result, err := ext.FilterArgs(&args)
			if err != nil {
				t.Fatal(err)
			}
			if !tc.error && len(result.Error) != 0 {
				t.Errorf("Unexpected error: %s\n", result.Error)
			}
			for _, failedNode := range tc.failedNodes {
				if _, exists := result.FailedNodes[failedNode]; !exists {
					t.Errorf("Expected that node %s will be invalidated: %v", failedNode, result.FailedNodes)
				}
			}
		})
	}
}
