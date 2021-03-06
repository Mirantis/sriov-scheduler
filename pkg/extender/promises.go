package extender

import (
	"fmt"
	"log"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
)

const (
	defaultPromisesCleanerInterval = 5 * time.Second
)

type PromisesInterface interface {
	PurgePromise(types.UID)
	MakePromise(types.UID)
	PromisesCount() *resource.Quantity
	Subscribe(chan struct{})
	RunPromisesCleaner(time.Duration, <-chan struct{})
}

func NewPromises() PromisesInterface {
	return &Promises{
		promises:    map[types.UID]time.Time{},
		subscribers: make([]chan struct{}, 0, 1),
	}
}

type Promises struct {
	sync.Mutex
	promises    map[types.UID]time.Time
	subscribers []chan struct{}
}

func (p *Promises) MakePromise(uid types.UID) {
	p.Lock()
	defer p.Unlock()
	log.Printf("promise made for %s\n", uid)
	p.promises[uid] = time.Now()
}

func (p *Promises) PurgePromise(uid types.UID) {
	p.Lock()
	defer p.Unlock()
	p.purgePromise(uid)
}

func (p *Promises) purgePromise(uid types.UID) {
	if _, exists := p.promises[uid]; !exists {
		return
	}
	delete(p.promises, uid)
	for _, s := range p.subscribers {
		close(s)
	}
	p.subscribers = make([]chan struct{}, 0, 1)
}

func (p *Promises) PromisesCount() *resource.Quantity {
	p.Lock()
	defer p.Unlock()
	log.Printf("promises count %d\n", len(p.promises))
	return resource.NewQuantity(int64(len(p.promises)), resource.DecimalSI)
}

func (p *Promises) Subscribe(waitChan chan struct{}) {
	p.Lock()
	defer p.Unlock()
	p.subscribers = append(p.subscribers, waitChan)
}

func (p *Promises) RunPromisesCleaner(interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			fmt.Println("Purging promises.")
			p.purgePromises(time.Now())
		case <-stopCh:
			return
		}
	}
}

func (p *Promises) purgePromises(fromTime time.Time) {
	p.Lock()
	defer p.Unlock()
	for podUID, promise := range p.promises {
		if promise.Sub(fromTime).Seconds() >= (10 * time.Second).Seconds() {
			p.purgePromise(podUID)
		}
	}
}
