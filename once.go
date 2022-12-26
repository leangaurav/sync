package sync

import (
	"sync"
	"sync/atomic"
)

// The simplest use of Once looks like this
// NewOnce(false, f)
//    and f is the function that should be called once

type FuncType func()

type Once interface {
	// Do function is used to execute the function/s once.
	// If multiple goroutines call Do(), only one goroutine will succede and remaining routines will block till one finishes.
	// The go-routine which successfully calls the function/s will receive `true` from Do(), others get a `false`. Returns true even if one of the functions panics.
	// This helps identify which call to Do() was successful if there are mulitple and the client needs to know which one worked.
	Do() bool

	// Done returns if the Once is in DONE state
	// If Done() if called concurrently with Do() it will return false if Do() is still executing.
	// Returns true if any of the functions panics.
	//
	// Done(true) : blocking call. waits till the state becomes DONE or Close(). returns whether state is DONE or not.
	// Done(false) : returns immediately and returns whether state is DONE or not.
	Done(block bool) bool

	// Close() unblocks all goroutines waiting on Done(true)
	Close()
}

// Once defines the stateful type. Clients should use NewOnce to create objects
type once struct {
	mu            sync.Mutex
	fs            []FuncType
	done          uint32
	suppressPanic bool
	unblockCond   *sync.Cond // used to signal any blocking client about change of state
	unblock       uint32
}

// NewOnce returns a new Once object
// Requires atleast one callable function. Muliple functions are executed in order they were passed.
func NewOnce(suppressPanic bool, f FuncType, fs ...FuncType) Once {
	fs = append([]FuncType{f}, fs...)

	return &once{
		mu:            sync.Mutex{},
		fs:            fs,
		suppressPanic: suppressPanic,
		unblockCond:   sync.NewCond(&sync.Mutex{}),
		unblock:       0,
	}
}

func (d *once) Do() (ret bool) {
	// fast path: if already done, no need to lock
	if atomic.LoadUint32(&d.done) == 1 {
		return
	}

	if d.suppressPanic {
		defer func() {
			recover()
		}()
	}

	// slow path: lock and call function once
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.done == 1 {
		return
	}

	// signal all waiting goroutines
	defer d.unblockCond.Broadcast()
	defer atomic.StoreUint32(&d.done, 1)
	ret = true // ensure true even if f() does panic
	for _, f := range d.fs {
		f()
	}

	return true
}

func (d *once) Done(block bool) bool {

	// blocking behavior
	if block {
		for atomic.LoadUint32(&d.unblock) == 0 {
			if atomic.LoadUint32(&d.done) == 1 {
				break
			}
			d.unblockCond.L.Lock()
			d.unblockCond.Wait()
			d.unblockCond.L.Unlock()
		}
	}

	return atomic.LoadUint32(&d.done) == 1
}

func (d *once) Close() {
	atomic.StoreUint32(&d.unblock, 1)
	d.unblockCond.Broadcast()
}
