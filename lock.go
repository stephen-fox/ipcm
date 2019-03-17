package lock

import (
	"fmt"
	"strings"
	"time"
)

const (
	configureErrPrefix    = "failed to configure lock -"
	unableToCreatePrefix  = "failed to create lock -"
	unableToAcquirePrefix = "failed to acquire lock -"
)

// Mutex is (...).
type Mutex interface {
	Lock()

	TryLock() error

	TimedTryLock(time.Duration) error

	Unlock()
}

// Acquirer is used to configure and acquire a Mutex.
type Acquirer interface {
	// SetResource sets the Mutex's resource. A "resource" is an object
	// that exists outside of the application which can be used as a mutex.
	//
	// New instances of an application must use the same argument
	// when acquiring the Mutex.
	//
	// On unix systems, this must be a string representing a fully qualified
	// file path.
	// For example:
	// 	/var/myapplication/lock
	//
	// On Windows, this is a string representing the name of a mutex object.
	// The string can consist of any character except backslash. For more
	// information, refer to the 'CreateMutexW' API documentation:
	// https://docs.microsoft.com/en-us/windows/desktop/api/synchapi/nf-synchapi-createmutexw
	// For example:
	// 	myapplication
	SetResource(string) Acquirer

	// Acquire acquires the Mutex. A non-nil error is returned
	// if the Mutex cannot be acquired.
	//
	// Be advised that Windows requires the Mutex be released by the
	// same thread that originally acquired the Mutex. Please review
	// 'runtime.LockOSThread()' for more information.
	//
	// The following defaults are used if not specified:
	// 	Acquire timeout: 2 seconds
	Acquire() (Mutex, error)
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
