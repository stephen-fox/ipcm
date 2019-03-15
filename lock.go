package lock

import (
	"fmt"
	"strings"
	"time"
)

const (
	acquireTimeout        = 2 * time.Second
	configureErrPrefix    = "failed to configure lock -"
	unableToCreatePrefix  = "failed to create lock -"
	unableToAcquirePrefix = "failed to acquire lock -"
)

// Lock represents a single instance of a running application.
type Lock interface {
	// Release releases the Lock.
	//
	// Be advised that Windows requires the Lock be released by the
	// same thread that originally acquired the Lock. Please review
	// 'runtime.LockOSThread()' for more information.
	Release() error
}

// Acquirer is used to configure and acquire a Lock.
type Acquirer interface {
	// SetAcquireTimeout sets the amount of time to wait when acquiring
	// the Lock.
	SetAcquireTimeout(time.Duration) Acquirer

	// SetResource sets the Lock's resource. A "resource" is an object
	// that exists outside of the application which can be used as a mutex.
	//
	// New instances of an application must use the same argument
	// when acquiring the Lock.
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

	// Acquire acquires the Lock. A non-nil error is returned
	// if the Lock cannot be acquired.
	//
	// Be advised that Windows requires the Lock be released by the
	// same thread that originally acquired the Lock. Please review
	// 'runtime.LockOSThread()' for more information.
	//
	// The following defaults are used if not specified:
	// 	Acquire timeout: 2 seconds
	Acquire() (Lock, error)
}

type defaultAcquirer struct {
	acquireTimeout time.Duration
	resource       string
	unexpectedLoss chan error
}

func (o *defaultAcquirer) SetAcquireTimeout(timeout time.Duration) Acquirer {
	o.acquireTimeout = timeout
	return o
}

func (o *defaultAcquirer) SetResource(resource string) Acquirer {
	o.resource = resource
	return o
}

func (o *defaultAcquirer) validateCommon() error {
	if o.acquireTimeout == 0 {
		o.acquireTimeout = acquireTimeout
	}

	if len(strings.TrimSpace(o.resource)) == 0 {
		return &ConfigureError{
			reason:     fmt.Sprintf("%s a well known resource was not specified",
				configureErrPrefix),
			noResource: true,
		}
	}

	return nil
}

func NewAcquirer() Acquirer {
	return &defaultAcquirer{}
}
