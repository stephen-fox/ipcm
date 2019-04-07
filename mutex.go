package lock

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	configureErrPrefix    = "failed to configure mutex -"
	unableToCreatePrefix  = "failed to create mutex -"
	unableToAcquirePrefix = "failed to acquire mutex -"

	infiniteOsMutexLockTimeout time.Duration = -1
)

// Mutex is a thread-safe object that functions in a similar manner to
// sync.Mutex, only it works across process boundaries. It can be used to
// orchestrate the execution of threads between different processes in the
// same way that a sync.Mutex controls access to data between multiple
// go routines.
//
// In other words, when one routine locks the mutex, all other routines within
// the current process and external process(es) will block if they attempt to
// lock the mutex.
type Mutex interface {
	// Lock locks the mutex.
	//
	// Be advised that any underlying errors that occur when locking
	// the OS mutex are hidden from the caller when using this method.
	// The underlying implementation of Lock() involves communicating
	// with external system APIs that *can* fail. In the event of an
	// API failure, the Lock() call will continue to retry until it
	// is successful.
	Lock()

	// TimedTryLock attempts to lock the Mutex within the specified
	// timeout. A non-nil error is returned when the Mutex cannot be
	// locked in time.
	TimedTryLock(time.Duration) error

	// Unlock unlocks the Mutex. Like sync.Mutex, the method will panic
	// if the Mutex is already unlocked.
	Unlock()
}

func validateResourceCommon(resource string) error {
	if len(strings.TrimSpace(resource)) == 0 {
		return &ConfigureError{
			reason:     fmt.Sprintf("%s a well known resource was not specified",
				configureErrPrefix),
			noResource: true,
		}
	}

	return nil
}

// timedSyncMutexLock attempts to lock the supplied *sync.Mutex within the
// specified timeout. If successful, the function returns the remaining
// timeout. A non-nil error is returned if the lock attempt exceeds
// the timeout.
func timedSyncMutexLock(mutex *sync.Mutex, timeout time.Duration) (time.Duration, error) {
	start := time.Now()
	mutexControl := make(chan struct{})

	go func() {
		mutex.Lock()

		select {
		case <-mutexControl:
			// Channel is closed - the main routine has
			// given up.
			mutex.Unlock()
		default:
			// Channel is still open, let the main routine know
			// that we locked the mutex.
			mutexControl <- struct{}{}
			// See what the main routine wants to do.
			_, open := <-mutexControl
			if !open {
				// The main routine closed the channel because it
				// gave up.
				mutex.Unlock()
			}
		}
	}()

	hitTimeout := time.NewTimer(timeout)

	select {
	case <-hitTimeout.C:
		close(mutexControl)
		return 0, &AcquireError{
			reason: fmt.Sprintf("%s *sync.Mutex lock attempt exceeded timeout of %s",
				unableToAcquirePrefix, timeout.String()),
			// TODO: bool.
		}
	case <-mutexControl:
		hitTimeout.Stop()
		// TODO: Maybe reverse this?
		mutexControl <- struct{}{}
		// TODO: What happens if timeout == 0?
		return timeout - time.Since(start), nil
	}
}
