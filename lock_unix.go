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

	err = os.Remove(o.file.Name())
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

	if !path.IsAbs(o.location) || len(o.location) == 1 {
		return nil, &ConfigureError{
			reason: configureErrPrefix + "the specified location is not a fully qualified file path - '" +
				o.location + "'",
			notAbs: true,
		}
	}

	err = os.MkdirAll(path.Dir(o.location), dirMode)
	if err != nil {
		return nil, &AcquireError{
			reason:  unableToCreatePrefix + err.Error(),
			dirFail: true,
		}
	}

	f, err := os.OpenFile(o.location, os.O_RDONLY|os.O_CREATE, lockMode)
	if err != nil {
		return nil, &AcquireError{
			reason:     unableToCreatePrefix + err.Error(),
			createFail: true,
		}
	}

	err = timedFlock(f, o.acquireTimeout)
	if err != nil {
		f.Close()
		return nil, &AcquireError{
			reason:     fmt.Sprintf("%stried to get lock for %s - %s",
				unableToAcquirePrefix, o.acquireTimeout.String(), err.Error()),
			createFail: true,
		}
	}

	return  &unixLock{
		mutex: &sync.Mutex{},
		file:  f,
	}, nil
}

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
