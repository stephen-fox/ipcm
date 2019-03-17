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

type unixMutex struct {
	mutex *sync.Mutex
	file  *os.File
	path  string
}

func (o *unixMutex) Lock() {
	o.mutex.Lock()

	if !o.lockUnsafe(-1) {
		o.mutex.Unlock()
	}
}

func (o *unixMutex) TryLock() error {
	o.mutex.Lock()

	if _, statErr := o.file.Stat(); statErr != nil {
		err := o.resetUnsafe()
		if err != nil {
			o.mutex.Unlock()
			return err
		}
	}

	err := o.resetUnsafe()
	if err != nil {
		o.mutex.Unlock()
		return err
	}

	err = unix.Flock(int(o.file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		o.mutex.Unlock()
		return err
	}

	return nil
}

func (o *unixMutex) TimedTryLock(timeout time.Duration) error {
	start := time.Now()
	locked := make(chan struct{})
	unlock := make(chan bool)

	go func() {
		o.mutex.Lock()
		locked <- struct{}{}
		if <-unlock {
			o.mutex.Unlock()
		}
	}()

	hitTimeout := time.After(timeout)

	select {
	case <-hitTimeout:
		break
	case <-locked:
		elapsed := time.Since(start)
		if elapsed < timeout && o.lockUnsafe(timeout - elapsed) {
			return nil
		}
	}

	go func() {
		unlock <- true
	}()

	return &AcquireError{
		reason: fmt.Sprintf("%s lock took longer than %s",
			unableToAcquirePrefix, timeout.String()),
		// TODO: bool.
	}
}

func (o *unixMutex) lockUnsafe(timeout time.Duration) bool {
	start := time.Now()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		if timeout > 0 && time.Since(start) >= timeout {
			return false
		}

		if _, statErr := o.file.Stat(); statErr != nil {
			err := o.resetUnsafe()
			if err != nil {
				<-ticker.C
				continue
			}
		}

		flockErr := unix.Flock(int(o.file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if flockErr == nil {
			return true
		}

		<-ticker.C
	}
}

func (o *unixMutex) resetUnsafe() error {
	if o.file != nil {
		o.file.Close()
	}

	err := os.MkdirAll(path.Dir(o.path), dirMode)
	if err != nil {
		return &AcquireError{
			reason:  fmt.Sprintf("%s %s", unableToCreatePrefix, err.Error()),
			dirFail: true,
		}
	}

	o.file, err = os.OpenFile(o.path, os.O_RDONLY|os.O_CREATE, lockMode)
	if err != nil {
		return &AcquireError{
			reason:     fmt.Sprintf("%s %s", unableToCreatePrefix, err.Error()),
			createFail: true,
		}
	}

	return nil
}

func (o *unixMutex) Unlock() {
	o.mutex.Unlock()

	if o.file == nil {
		return
	}

	err := unix.Flock(int(o.file.Fd()), unix.LOCK_UN)
	if err != nil {
		return
	}
}

// NewMutex creates a new mutex using an object that exists outside of
// the application.
//
// New instances of an application must use the same argument
// when acquiring the Mutex.
//
// On unix systems, this must be a string representing a fully qualified
// file path.
// For example:
// 	/var/myapplication/lock
func NewMutex(resourcePath string) (Mutex, error) {
	err := validateResourceCommon(resourcePath)
	if err != nil {
		return nil, err
	}

	if !path.IsAbs(resourcePath) || len(resourcePath) == 1 {
		return nil, &ConfigureError{
			reason: fmt.Sprintf("%s the specified resource is not a fully qualified file path - '%s'",
				configureErrPrefix, resourcePath),
			notAbs: true,
		}
	}

	mu := &unixMutex{
		mutex: &sync.Mutex{},
		path:  resourcePath,
	}

	err = mu.resetUnsafe()
	if err != nil {
		return nil, err
	}

	return mu, nil
}
