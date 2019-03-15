// +build !windows

package lock

import (
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const (
	dirMode  = 0755
	lockMode = 0644
)

type unixLock struct {
	mutex *sync.Mutex
	file  *os.File
}

func (o *unixLock) Release() error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.file == nil {
		return nil
	}

	err := unix.Flock(int(o.file.Fd()), unix.LOCK_UN)
	if err != nil {
		return err
	}

	err = o.file.Close()
	if err != nil {
		return err
	}

	o.file = nil

	return nil
}

func (o *defaultAcquirer) Acquire() (Lock, error) {
	err := o.validateCommon()
	if err != nil {
		return nil, err
	}

	if !path.IsAbs(o.resource) || len(o.resource) == 1 {
		return nil, &ConfigureError{
			reason: fmt.Sprintf("%s the specified resource is not a fully qualified file path - '%s'",
				configureErrPrefix, o.resource),
			notAbs: true,
		}
	}

	err = os.MkdirAll(path.Dir(o.resource), dirMode)
	if err != nil {
		return nil, &AcquireError{
			reason:  fmt.Sprintf("%s %s", unableToCreatePrefix, err.Error()),
			dirFail: true,
		}
	}

	f, err := os.OpenFile(o.resource, os.O_RDONLY|os.O_CREATE, lockMode)
	if err != nil {
		return nil, &AcquireError{
			reason:     fmt.Sprintf("%s %s", unableToCreatePrefix, err.Error()),
			createFail: true,
		}
	}

	err = timedFlock(f, o.acquireTimeout)
	if err != nil {
		f.Close()
		return nil, &AcquireError{
			reason: fmt.Sprintf("%s tried to get lock for %s - %s",
				unableToAcquirePrefix, o.acquireTimeout.String(), err.Error()),
			inUse:  true,
		}
	}

	return  &unixLock{
		mutex: &sync.Mutex{},
		file:  f,
	}, nil
}

// TODO: This implementation can take longer than the specified timeout
//  because each attempt to flock takes some time.
func timedFlock(f *os.File, timeout time.Duration) error {
	var err error
	max := 10
	ticker := time.NewTicker(timeout / time.Duration(max))
	defer ticker.Stop()

	for i := 0; i < max; i++ {
		<-ticker.C
		err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return nil
		}
	}

	return err
}
