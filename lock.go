package lock

import (
	"strings"
	"time"
)

const (
	name                 = ".grundy-lock-41xJwGFewWhrYZje"
	acquireTimeout       = 2 * time.Second
	inUseErr             = "another instance of the application is already running"
	buildErrPrefix       = "failed to build lock - "
	unableToCreatePrefix = "failed to create lock - "
	unableToReadPrefix   = "failed to read lock - "
)

// Lock represents a single instance of a running application.
type Lock interface {
	// Acquire attempts to acquire control of the lock. If the lock
	// cannot be acquired, a non-nil error is returned.
	Acquire() error

	// Errs returns a chan for errors encountered while maintaining
	// the lock.
	Errs() chan error

	// Release releases control of the lock.
	Release()
}

type LockBuilder interface {
	// SetAcquireTimeout sets the amount of time to wait when acquiring
	// a lock.
	SetAcquireTimeout(time.Duration) LockBuilder

	// SetLocation sets the well-known location of the lock. New instances
	// of an application must use the same argument.
	//
	// On unix systems, the string must be a fully qualified file path.
	// For example:
	// 	/var/myapplication/lock
	//
	// On Windows, this must be a string that follows the Windows
	// PipeName rules:
	// 	"[The location string] can include any character
	// 	other than a backslash, including numbers and special
	// 	characters. The entire [location] string can be up to
	// 	256 characters long. [Location] names are
	// 	not case-sensitive."
	// 	https://docs.microsoft.com/en-us/windows/desktop/ipc/pipe-names
	// For example:
	//  myapplication-jdasjkldj84
	SetLocation(string) LockBuilder

	// Build generates a new instance of a Lock.
	//
	// The following defaults are used if not specified:
	// 	Acquire timeout: 2 seconds
	Build() (Lock, error)
}

type defaultLockBuilder struct {
	acquireTimeout time.Duration
	location       string
}

func (o *defaultLockBuilder) SetAcquireTimeout(timeout time.Duration) LockBuilder {
	o.acquireTimeout = timeout
	return o
}

func (o *defaultLockBuilder) SetLocation(location string) LockBuilder {
	o.location = location
	return o
}

func (o *defaultLockBuilder) validateCommon() error {
	if o.acquireTimeout == 0 {
		o.acquireTimeout = acquireTimeout
	}

	if len(strings.TrimSpace(o.location)) == 0 {
		return &BuildError{
			reason:     buildErrPrefix + "a well known location was not specified",
			noLocation: true,
		}
	}

	return nil
}

func NewLockBuilder() LockBuilder {
	return &defaultLockBuilder{}
}
