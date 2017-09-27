package extender

import (
	"strconv"
	"testing"

	"k8s.io/client-go/pkg/api/v1"
)

func TestNetworkSelector(t *testing.T) {
	testCases := []struct {
		networks string
		expected bool
	}{
		{
			networks: "sriov,contrail",
			expected: true,
		},
		{
			networks: "",
			expected: false,
		},
		{
			networks: "contrail",
			expected: false,
		},
		{
			networks: "sriov",
			expected: true,
		},
		{
			networks: "sriov,sriov,sriov",
			expected: true,
		},
	}
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			pod := &v1.Pod{}
			pod.SetAnnotations(map[string]string{"networks": tc.networks})
			if result := NetworkSelector(pod); result != tc.expected {
				t.Errorf("Expected result %v is different from received %v for networks %s",
					tc.expected, result, tc.networks)
			}
		})
	}

}
