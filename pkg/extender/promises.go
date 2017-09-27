package extender

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"
)

const (
	defaultPromisesCleanerInterval = 5 * time.Second
)

// promisesCleaner purges stale promises every interval
func (ext *Extender) RunPromisesCleaner(interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			fmt.Println("Purging promises.")
			ext.purgePromises(time.Now())
		case <-stopCh:
			return
		}
	}
}

func (ext *Extender) purgePromises(fromTime time.Time) {
	ext.Lock()
	defer ext.Unlock()
	for podUID, promise := range ext.promises {
		if promise.Sub(fromTime).Seconds() >= (10 * time.Second).Seconds() {
			delete(ext.promises, podUID)
			ext.promisedVFs.Sub(*singleItem)
		}
	}
}

func (ext *Extender) purgeByUID(uid types.UID) {
	if _, exists := ext.promises[uid]; !exists {
		return
	}
	delete(ext.promises, uid)
	ext.promisedVFs.Sub(*singleItem)
}
