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
	mutex  *sync.Mutex
	file   *os.File
	config MutexConfig
}

func (o *unixMutex) Lock() {
	o.mutex.Lock()

	o.lockOsMutexUnsafe(-1)
}

func (o *unixMutex) TimedTryLock(timeout time.Duration) error {
	remaining, err := timedSyncMutexLock(o.mutex, timeout)
	if err != nil {
		return err
	}

	if o.lockOsMutexUnsafe(remaining) {
		return nil
	}

	o.mutex.Unlock()

	return &AcquireError{
		reason:        fmt.Sprintf("%s system flock took longer than %s",
			unableToAcquirePrefix, timeout.String()),
		systemTimeout: true,
	}
}

func (o *unixMutex) lockOsMutexUnsafe(timeout time.Duration) bool {
	start := time.Now()
	sleep := 100 * time.Millisecond

	for {
		if timeout > 0 && time.Since(start) >= timeout {
			return false
		}

		if _, statErr := o.file.Stat(); statErr != nil {
			err := o.resetFileUnsafe()
			if err != nil {
				time.Sleep(sleep)
				continue
			}
		}

		flockErr := unix.Flock(int(o.file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if flockErr == nil {
			return true
		}

		time.Sleep(sleep)
	}
}

func (o *unixMutex) resetFileUnsafe() error {
	if o.file != nil {
		o.file.Close()
	}

	err := os.MkdirAll(path.Dir(o.config.Resource), dirMode)
	if err != nil {
		return &AcquireError{
			reason:  fmt.Sprintf("%s %s", unableToCreatePrefix, err.Error()),
			dirFail: true,
		}
	}

	o.file, err = os.OpenFile(o.config.Resource, os.O_RDONLY|os.O_CREATE, lockMode)
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

// NewMutex creates a new Mutex.
func NewMutex(config MutexConfig) (Mutex, error) {
	err := config.validate()
	if err != nil {
		return nil, err
	}

	if !path.IsAbs(config.Resource) || len(config.Resource) == 1 {
		return nil, &ConfigureError{
			reason: fmt.Sprintf("%s the specified resource is not a fully qualified file path - '%s'",
				configureErrPrefix, config.Resource),
			notAbs: true,
		}
	}

	mu := &unixMutex{
		mutex:  &sync.Mutex{},
		config: config,
	}

	err = mu.resetFileUnsafe()
	if err != nil {
		return nil, err
	}

	return mu, nil
}
