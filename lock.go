package lock

import (
	"strings"
	"time"
)

const (
	name                  = ".grundy-lock-41xJwGFewWhrYZje"
	acquireTimeout        = 2 * time.Second
	inUseErr              = "another instance of the application is already running"
	configureErrPrefix    = "failed to configure lock - "
	unableToCreatePrefix  = "failed to create lock - "
	unableToAcquirePrefix = "failed to acquire lock - "
)

// Lock represents a single instance of a running application.
type Lock interface {
	// Release releases control of the lock.
	Release() error
}

// Acquirer is used to configure and acquire a Lock.
type Acquirer interface {
	// SetAcquireTimeout sets the amount of time to wait when acquiring
	// the Lock.
	SetAcquireTimeout(time.Duration) Acquirer

	// SetLocation sets the well-known location of the Lock.
	// New instances of an application must use the same argument
	// when acquiring the Lock.
	//
	// On unix systems, the string must be a fully qualified file path.
	// For example:
	// 	/var/myapplication/lock
	//
	// On Windows, the string must follow the Windows PipeName rules:
	// 	"[The location string] can include any character
	// 	other than a backslash, including numbers and special
	// 	characters. The entire [location] string can be up to
	// 	256 characters long. [Location] names are
	// 	not case-sensitive."
	// 	https://docs.microsoft.com/en-us/windows/desktop/ipc/pipe-names
	// For example:
	//  myapplication-jdasjkldj84
	SetLocation(string) Acquirer

	// SetUnexpectedLossChan sets a channel that is notified when the
	// Lock is unexpectedly lost.
	SetUnexpectedLossChan(chan error) Acquirer

	// Acquire acquires the Lock. A non-nil error is returned
	// if the Lock cannot be acquired.
	//
	// The following defaults are used if not specified:
	// 	Acquire timeout: 2 seconds
	Acquire() (Lock, error)
}

type defaultAcquirer struct {
	acquireTimeout time.Duration
	location       string
	unexpectedLoss chan error
}

func (o *defaultAcquirer) SetAcquireTimeout(timeout time.Duration) Acquirer {
	o.acquireTimeout = timeout
	return o
}

func (o *defaultAcquirer) SetLocation(location string) Acquirer {
	o.location = location
	return o
}

func (o *defaultAcquirer) SetUnexpectedLossChan(c chan error) Acquirer {
	o.unexpectedLoss = c
	return o
}

func (o *defaultAcquirer) validateCommon() error {
	if o.acquireTimeout == 0 {
		o.acquireTimeout = acquireTimeout
	}

	if len(strings.TrimSpace(o.location)) == 0 {
		return &ConfigureError{
			reason:     configureErrPrefix + "a well known location was not specified",
			noLocation: true,
		}
	}

	return nil
}

func NewAcquirer() Acquirer {
	return &defaultAcquirer{}
}
