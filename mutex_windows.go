package lock

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	kernel32Name        = "kernel32.dll"
	createMutexW        = "CreateMutexW"
	releaseMutex        = "ReleaseMutex"
	waitForSingleObject = "WaitForSingleObject"
	globalPrefix        = "Global\\"
)

// TODO: Close handle?
type windowsMutex struct {
	config      MutexConfig
	mutex       *sync.Mutex
	winMutexApi *windowsMutexApi
	mutexHandle uintptr
}

func (o *windowsMutex) Lock() {
	o.mutex.Lock()

	o.lockOsMutexUnsafe(infiniteOsMutexLockTimeout)
}

func (o *windowsMutex) TimedTryLock(timeout time.Duration) error {
	remaining, err := timedSyncMutexLock(o.mutex, timeout)
	if err != nil {
		return err
	}

	err = o.lockOsMutexUnsafe(remaining)
	if err != nil {
		o.mutex.Unlock()
		return err
	}

	return nil
}

func (o *windowsMutex) lockOsMutexUnsafe(timeout time.Duration) error {
	start := time.Now()

	// TODO: Global should be an OS specific option.
	// TODO: Should this be stored in the object as a field?
	mutexId := uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(globalPrefix + o.config.Resource)))

	mutexHandle, _, err := o.winMutexApi.createMutex.Call(0, 0, mutexId)
	createMutexErrNum := int(err.(windows.Errno))
	switch err.(windows.Errno) {
	case 0, windows.ERROR_ALREADY_EXISTS:
		// If the mutex already exists, the Windows API still returns
		// a handle to the mutex.
		break
	default:
		if timeout == infiniteOsMutexLockTimeout {
			return o.lockOsMutexUnsafe(timeout)
		}
		return &AcquireError{
			reason:     fmt.Sprintf("%s got return code %d - %s",
				unableToCreatePrefix, createMutexErrNum, err.Error()),
			createFail: true,
		}
	}

	// Per the 'WaitForSingleObject' Windows API doc, the waitResult will
	// be a non-zero value if a failure occurs. Therefore, we can treat
	// the waitResult as an error condition. This appears to be a break
	// in the Windows API pattern:
	//  https://docs.microsoft.com/en-us/windows/desktop/api/synchapi/nf-synchapi-waitforsingleobject#return-value
	waitResult, _, err := o.winMutexApi.waitForSingleObject.Call(mutexHandle, uintptr(timeout.Seconds() * 1000))
	if timeout == infiniteOsMutexLockTimeout && waitResult != 0 {
		return o.lockOsMutexUnsafe(infiniteOsMutexLockTimeout)
	}

	switch waitResult {
	case windows.WAIT_OBJECT_0:
		o.mutexHandle = mutexHandle
		return nil
	case windows.WAIT_ABANDONED:
		return o.lockOsMutexUnsafe(timeout - time.Since(start))
	case windows.WAIT_TIMEOUT:
		return &AcquireError{
			reason:        fmt.Sprintf("%s exceeded wait timeout of %s",
				unableToAcquirePrefix, timeout.String()),
			systemTimeout: true,
		}
	case windows.WAIT_FAILED:
		waitForErrNum := int(err.(windows.Errno))
		if waitForErrNum != 0 {
			return &AcquireError{
				reason: fmt.Sprintf("%s got return code %d - %s",
					unableToAcquirePrefix, waitForErrNum, err.Error()),
				inUse:  true,
			}
		}
	}

	return &AcquireError{
		reason: fmt.Sprintf("%s system mutex wait failed, got return code %d",
			unableToAcquirePrefix, waitResult),
		inUse:  true,
	}
}

func (o *windowsMutex) Unlock() {
	o.unlockUnsafe()

	o.mutex.Unlock()
}

func (o *windowsMutex) unlockUnsafe() error {
	_, _, err := o.winMutexApi.release.Call(o.mutexHandle)
	errNum := int(err.(windows.Errno))
	if errNum > 0 {
		return fmt.Errorf("got return code %d - %s", errNum, err.Error())
	}

	return nil
}

type windowsMutexApi struct {
	kernel32            *windows.LazyDLL
	createMutex         *windows.LazyProc
	waitForSingleObject *windows.LazyProc
	release             *windows.LazyProc
}

// NewMutex creates a new Mutex.
//
// Be advised that Windows requires the Mutex be unlocked or released by the
// same thread that originally locked the Mutex. Please review
// 'runtime.LockOSThread()' for more information.
func NewMutex(config MutexConfig) (Mutex, error) {
	err := config.validate()
	if err != nil {
		return nil, err
	}

	winApi, err := loadWindowsMutexApi()
	if err != nil {
		return nil, err
	}

	mu := &windowsMutex{
		mutex:       &sync.Mutex{},
		config:      config,
		winMutexApi: winApi,
	}

	return mu, nil
}

func loadWindowsMutexApi() (*windowsMutexApi, error) {
	kernel32 := windows.NewLazyDLL(kernel32Name)
	if kernel32 == nil {
		return nil, &AcquireError{
			reason:  fmt.Sprintf("%s failed to load %s",
				unableToCreatePrefix, kernel32Name),
			dllFail: true,
		}
	}

	createMutexProc, err := getProcedure(createMutexW, kernel32)
	if err != nil {
		return nil, &AcquireError{
			reason:   fmt.Sprintf("%s %s",
				unableToCreatePrefix, err.Error()),
			procFail: true,
		}
	}

	waitForSingleObjectProc, err := getProcedure(waitForSingleObject, kernel32)
	if err != nil {
		return nil, &AcquireError{
			reason:   fmt.Sprintf("%s - %s",
				unableToCreatePrefix, err.Error()),
			procFail: true,
		}
	}

	releaseMutexProc, err := getProcedure(releaseMutex, kernel32)
	if err != nil {
		return nil, &AcquireError{
			reason:   fmt.Sprintf("%s - %s",
				unableToCreatePrefix, err.Error()),
			procFail: true,
		}
	}

	return &windowsMutexApi{
		kernel32:            kernel32,
		createMutex:         createMutexProc,
		waitForSingleObject: waitForSingleObjectProc,
		release:             releaseMutexProc,
	}, nil
}

func getProcedure(procedureName string, dll *windows.LazyDLL) (*windows.LazyProc, error) {
	proc := dll.NewProc(procedureName)
	if proc == nil {
		return nil, fmt.Errorf("procedure %s in DLL %s is nil",
			procedureName, dll.Name)
	}

	err := proc.Find()
	if err != nil {
		return nil, err
	}

	return proc, nil
}
