package lock

import (
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
