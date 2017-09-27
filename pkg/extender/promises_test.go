package extender

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
)

func TestPromisesCleaner(t *testing.T) {
	ext := NewExtender(nil)

	invalidPromise := time.Now().Add(11 * time.Second)
	validPromise := time.Now()
	ext.promises = map[types.UID]time.Time{
		types.UID("1"): invalidPromise,
		types.UID("2"): invalidPromise,
		types.UID("3"): validPromise,
	}
	ext.promisedVFs.Add(*resource.NewQuantity(3, resource.DecimalSI))
	ext.purgePromises(time.Now())
	if len(ext.promises) != 1 {
		t.Errorf("Only one promise is valid: %v", ext.promises)
	}
}
