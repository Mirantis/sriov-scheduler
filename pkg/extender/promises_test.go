package extender

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/types"
)

func TestPromisesCleaner(t *testing.T) {
	p := &Promises{
		promises:    map[types.UID]time.Time{},
		subscribers: make([]chan struct{}, 0, 1),
	}
	invalidPromise := time.Now().Add(11 * time.Second)
	validPromise := time.Now()
	p.promises = map[types.UID]time.Time{
		types.UID("1"): invalidPromise,
		types.UID("2"): invalidPromise,
		types.UID("3"): validPromise,
	}
	p.purgePromises(time.Now())
	if len(p.promises) != 1 {
		t.Errorf("Only one promise is valid: %v", p.promises)
	}
}
