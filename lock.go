package lock

import (
	"strings"
	"time"
)

const (
	name                 = ".grundy-lock-41xJwGFewWhrYZje"
	acquireTimeout       = 2 * time.Second
	inUseErr             = "another instance of the application is already running"
	unableToCreatePrefix = "failed to create lock - "
	unableToReadPrefix   = "failed to read lock - "
)

type Lock interface {
	Acquire() error
	Errs() chan error
	Release()
}

type LockBuilder interface {
	// SetAcquireTimeout sets the amount of time to wait when acquiring
	// a lock.
	SetAcquireTimeout(time.Duration) LockBuilder

	// SetParentDirPath sets the parent directory o
	SetParentDirPath(string) LockBuilder
	SetName(string) LockBuilder
	Build() (Lock, error)
}

type defaultLockBuilder struct {
	acquireTimeout time.Duration
	parentDirPath  string
	name           string
}

func (o *defaultLockBuilder) SetAcquireTimeout(timeout time.Duration) LockBuilder {
	o.acquireTimeout = timeout
	return o
}

func (o *defaultLockBuilder) SetParentDirPath(dirPath string) LockBuilder {
	o.parentDirPath = dirPath
	return o
}

func (o *defaultLockBuilder) SetName(name string) LockBuilder {
	o.name = name
	return o
}

func (o *defaultLockBuilder) validateCommon() error {
	if o.acquireTimeout == 0 {
		o.acquireTimeout = acquireTimeout
	}

	if len(strings.TrimSpace(o.name)) == 0 {

	}
}
