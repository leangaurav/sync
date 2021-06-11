package sync

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// The simplest use of Once looks like this
// NewOnce(false, false, sync.VerifyNone, f)
//   lazyDone = false, suppressPanic = false, verify = VerifyNone
//    and f is the function that should be called once. f is of type `func ()bool`
// The above setup creates a Once object.
// lazyDone=false => On calling the Do() method of the Once object for the first time
//   first the value of done is set and then the function f() is invoked
//

// FuncType all functions passed to Once should have this signature.
// i.e. they don't accept arguments and always return a bool.
// You need to wrap you function is it's doesn't staisfy FuncType.
// The return value of functions is used by Once based on VerifyType.
type FuncType func() bool

// VerifyType decides how Do will interpret and use the return values of the functions.
// The default is VerifyNone i.e. return values are not considered while setting state to DONE.
type VerifyType string

const (
	VerifyNone        VerifyType = ""                  // No verification of return value/s: Default
	VerifyAll         VerifyType = "VerifyAll"         // the Once is set in DONE state only if all functions return true
	VerifyFirstRunAll VerifyType = "VerifyFirstRunAll" // the Once is set in DONE state based on first function returning true, but all functions are executed
	VerifyFirstExit   VerifyType = "VerifyFirstExit"   // Set state to DONE based on the first function that returns true. Skip execution of remaining functions
)

// Once defines the stateful type. Clients should use NewOnce to create objects
type Once struct {
	mu             sync.Mutex
	fs             []FuncType
	done           uint32
	lazyDone       bool
	suppressPanic  bool
	doneFromVerify bool
	verify         VerifyType
	unblockCond    *sync.Cond // used to signal any blocking client about change of state
	unblock        uint32
}

// NewDefaultOnce is a wrapper over NewOnce. It returns a Once object with the default options.
func NewDefaultOnce(f FuncType, fs ...FuncType) (*Once, error) {
	return NewOnce(false, false, VerifyNone, f, fs...)
}

// NewOnce returns a new Once object with the give options. Atleast one function needs to be given.
// In case of muliple functions, they are executed in the order they were passed.
//
// Below paramether combinations will raise error:
//   - lazyDone = true; verify = VerifyAll / VerifyFirstRunAll / VerifyFirstExit
func NewOnce(lazyDone bool, suppressPanic bool, verify VerifyType, f FuncType, fs ...FuncType) (*Once, error) {
	fs = append([]FuncType{f}, fs...)

	if lazyDone == false && verify != VerifyNone {
		return nil, fmt.Errorf("lazyDone needs to true when using verify=%s or set verify=%s", verify, VerifyNone)
	}

	return &Once{
		mu:            sync.Mutex{},
		fs:            fs,
		lazyDone:      lazyDone,
		suppressPanic: suppressPanic,
		verify:        verify,
		unblockCond:   sync.NewCond(&sync.Mutex{}),
		unblock:       0,
	}, nil
}

// Do function is used to execute the function/s once.
// When multiple goroutines try to call Do(), only one go-routine will be able to call the function/s at a time.
// The remaining go-routines stay blocked till the first one finishes.
// The go-routine which is able to successfully call the function/s will get `true` returned by Do(), all others get a false.
// This helps identify which call to Do() was successful if there are mulitple and the client needs to know which one worked.
// A lot of what Do ends up doing will depend on the different options used while crating Once.
//
//   lazyDone bool
// when lazyDone = false, Do() firt sets the state as DONE and then goes on to execute the function/s.
// If lazyDone = false, Do() first calls function and then sets DONE. Whether DONE gets set is also dependent on Verify options.
//
//   suppressPanic bool
// if suppressPanic = true, any panics from the code executed by function/s will be suppressed.
// When panics are suppressed, successive Do() calls may or may not trigger the function/s even if the first exection did panic.
// That is dependent on the value of Verify used. See the unit test cases for all possible cases
func (d *Once) Do() (res bool) {
	res = false
	// fast path: if already done, no need to lock
	if atomic.LoadUint32(&d.done) == 1 {
		return false
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
		return false
	}

	// signal all waiting goroutines
	defer d.unblockCond.Broadcast()

	// check if done needs to be set before or after calling the function
	if d.lazyDone == false {
		atomic.StoreUint32(&d.done, 1)
	}

	if d.verify == VerifyAll {

		tempRes := true
		for _, f := range d.fs {
			tempRes = tempRes && f()
		}
		res = tempRes

	} else if d.verify == VerifyFirstRunAll {

		for _, f := range d.fs {
			res = f() || res // f() should be the first arg to || operator
		}

	} else if d.verify == VerifyFirstExit {

		for _, f := range d.fs {
			if f() {
				res = true
				break
			}
		}

	} else {

		res = true
		for _, f := range d.fs {
			f()
		}

	}

	if d.lazyDone == true && res {
		atomic.StoreUint32(&d.done, 1)
	}

	return res
}

// Done returns if the Once is in DONE state. Calls to Done() are non-blocking.
// Value used for lazyDone changes behavior in case of concurrent access.
// If Done() if called concurrently with Do() it may return true even if Do() is still executing.
//
// Done(true) : the calling goroutines will block till the state becomes DONE or Close() is called explicitly to unblocak all goroutines
//              returns true or false based on whether state is DONE or not.
// Done(false) : returns immediately and returns whether state is DONE or not.
func (d *Once) Done(block bool) bool {

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

// Reset resets Once for reuse.
// It also returns if reset was actually required or not.
// Reset is concurrency safe. In case a Do() call is already in progress(has acquired the lock), reset will happen after Do() finishes.
func (d *Once) Reset() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	atomic.StoreUint32(&d.unblock, 0)
	res := atomic.LoadUint32(&d.done) == 1
	atomic.StoreUint32(&d.done, 0)
	return res
}

// Close() unblocks all goroutines waiting on Done(true)
func (d *Once) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	atomic.StoreUint32(&d.unblock, 1)
	d.unblockCond.Broadcast()
}
