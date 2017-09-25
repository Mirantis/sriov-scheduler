package extender

import (
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
)

func TestExperimentParseQuantity(t *testing.T) {
	mult, _ := resource.ParseQuantity("10")
	single, _ := resource.ParseQuantity("1")
	mult.Sub(single)
	fmt.Println(&mult)
}
