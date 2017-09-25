package extender

import (
	"fmt"
	"time"
)

const (
	defaultPromisesCleanerInterval = 5 * time.Second
)

// promisesCleaner purges stale promises every interval
func (ext *Extender) promisesCleaner(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		<-ticker.C
		fmt.Println("Purging promises.")
		ext.purgePromises()
	}
}

func (ext *Extender) purgePromises() {
	ext.Lock()
	defer ext.Unlock()
	for i, promise := range ext.promises {
		if time.Now().Sub(promise).Seconds() >= (10 * time.Second).Seconds() {
			copy(ext.promises[i:], ext.promises[i+1:])
			ext.promises = ext.promises[:len(ext.promises)-1]
		}
	}
}
