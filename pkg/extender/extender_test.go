package extender

import (
	"fmt"
	"strconv"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/pkg/api/v1"

	"sort"

	"github.com/stretchr/testify/require"
)

func TestFilter(t *testing.T) {
	testCases := []struct {
		nodesResources  []int64
		alreadyPromised int
		failedNodes     []string
		error           bool
	}{
		{
			nodesResources:  []int64{1, 1, 0},
			alreadyPromised: 1,
			failedNodes:     []string{"0", "1", "2"},
			error:           true,
		},
		{
			nodesResources:  []int64{2, 0},
			alreadyPromised: 1,
			failedNodes:     []string{"1"},
		},
		{
			nodesResources:  []int64{3, 3},
			alreadyPromised: 0,
		},
	}
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ext := NewExtender(nil)
			for j := 0; j < tc.alreadyPromised; j++ {
				ext.promises.MakePromise(types.UID(fmt.Sprintf("00%d", j)))
			}
			resultInterface, err := ext.FilterArgs(makeExtenderArgs(tc.nodesResources))
			if err != nil {
				t.Fatal(err)
			}
			result := resultInterface.(*ExtenderFilterResult)
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

func TestPrioritize(t *testing.T) {
	testCases := []struct {
		resources     []int64
		expectedOrder []string
	}{
		{
			resources:     []int64{10, 5, 0},
			expectedOrder: []string{"0", "1", "2"},
		},
		{
			resources:     []int64{0, 1, 2},
			expectedOrder: []string{"2", "1", "0"},
		},
	}
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ext := NewExtender(nil)
			priorities, err := ext.Prioritize(makeExtenderArgs(tc.resources))
			if err != nil {
				t.Fatal(err)
			}
			priorityList := *priorities.(*HostPriorityList)
			require.Len(t, priorityList, len(tc.expectedOrder))
			sort.Slice(priorityList, func(i, j int) bool {
				return priorityList[i].Score > priorityList[j].Score
			})
			for i, priority := range priorityList {
				require.Equal(t, priority.Host, tc.expectedOrder[i])
			}
		})
	}
}

func makeNode(i int, totalvfs int64) v1.Node {
	return v1.Node{
		ObjectMeta: meta_v1.ObjectMeta{Name: strconv.Itoa(i)},
		Status: v1.NodeStatus{
			Allocatable: v1.ResourceList{
				TotalVFsResource: *resource.NewQuantity(
					totalvfs, resource.DecimalSI)},
		}}
}

func makePod(uid string) v1.Pod {
	return v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			UID:         types.UID(uid),
			Annotations: map[string]string{"networks": "sriov"},
		}}
}

func makeExtenderArgs(resources []int64) *ExtenderArgs {
	args := ExtenderArgs{
		Pod:   makePod("first"),
		Nodes: &v1.NodeList{Items: []v1.Node{}},
	}
	for i, totalvfs := range resources {
		args.Nodes.Items = append(args.Nodes.Items, makeNode(i, totalvfs))
	}
	return &args
}
