// +build !windows

package lock

import (
	"bufio"
	"os"
	"path"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const (
	dirMode  = 0755
	pipeMode = 0644
)

type unixLock struct {
	Lock
	mutex    *sync.Mutex
	errs     chan error
	stop     chan chan struct{}
	location string
}

func (o *unixLock) Acquire() error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	select {
	case _, open := <-o.stop:
		if !open {
			o.stop = make(chan chan struct{})
		}
	default:
		return nil
	}

	err := os.MkdirAll(path.Dir(o.location), dirMode)
	if err != nil {
		return &AcquireError{
			reason:  err.Error(),
			dirFail: true,
		}
	}

	_, statErr := os.Stat(o.location)
	if statErr == nil {
		err := acquirePipe(o.location)
		if err != nil {
			close(o.stop)
			return err
		}
	} else {
		err := syscall.Mkfifo(o.location, pipeMode)
		if err != nil {
			close(o.stop)
			return &AcquireError{
				reason:     unableToCreatePrefix + err.Error(),
				createFail: true,
			}
		}
	}

	go o.manage()

	return nil
}

func (o *unixLock) manage() {
	done := make(chan struct{})

	go func() {
		for {
			f, err := os.OpenFile(o.location, os.O_WRONLY, pipeMode)
			select {
			case _, open := <-done:
				if !open {
					f.Close()
					return
				}
			default:
				if err != nil {
					o.errs <- err
					continue
				}

				_, err = f.WriteString(strconv.Itoa(os.Getpid()) + "\n")
				if err != nil {
					f.Close()
					o.errs <- err
					continue
				}

				f.Close()
			}
		}
	}()

	c := <-o.stop
	close(done)
	os.Remove(o.location)
	c <- struct{}{}
}

func (o *unixLock) Errs() chan error {
	return o.errs
}

func (o *unixLock) Release() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	select {
	case _, open := <-o.stop:
		if !open {
			return
		}
	default:
	}

	c := make(chan struct{})
	o.stop <- c
	<-c

	close(o.stop)
}

func acquirePipe(pipePath string) error {
	openResult := make(chan error)
	defer close(openResult)

	go func() {
		f, err := os.Open(pipePath)
		if err == nil {
			// Drain the pipe to prevent "broken pipe" errors
			// on the writer's end.
			scanner := bufio.NewScanner(f)
			scanner.Scan()
		}
		f.Close()

		select {
		case _, open := <-openResult:
			if !open {
				return
			}
		default:
			openResult <- err
		}
	}()

	timeout := time.NewTimer(acquireTimeout)

	select {
	case err := <-openResult:
		// Another instance of the application owns the pipe
		// if we can read before the timeout occurs.
		if err != nil {
			return &AcquireError{
				reason:   unableToReadPrefix + err.Error(),
				readFail: true,
			}
		}

		return &AcquireError{
			reason: inUseErr,
			inUse:  true,
		}
	case <-timeout.C:
		// No one is home.
	}

	return nil
}

func (o *defaultLockBuilder) Build() (Lock, error) {
	if !path.IsAbs(o.location) || len(o.location) == 1 {
		return &unixLock{}, &BuildError{
			reason: buildErrPrefix + "the specified location is not a fully qualified file path - '" + o.location + "'",
			notAbs: true,
		}
	}

	l := &unixLock{
		mutex:    &sync.Mutex{},
		errs:     make(chan error),
		stop:     make(chan chan struct{}),
		location: o.location,
	}

	close(l.stop)

	return l, nil
}
