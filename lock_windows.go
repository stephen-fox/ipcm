package lock

import (
	"fmt"
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

type windowsLock struct {
	kernel32    *windows.LazyDLL
	mutexHandle uintptr
}

func (o *windowsLock) Release() error {
	releaseMutexProc, err := getProcedure(releaseMutex, o.kernel32)
	if err != nil {
		return err
	}

	_, _, err = releaseMutexProc.Call(o.mutexHandle)
	errNum := int(err.(windows.Errno))
	if errNum > 0 {
		return fmt.Errorf("got return code %d - %s", errNum, err.Error())
	}

	err = windows.CloseHandle(windows.Handle(o.mutexHandle))
	if err != nil {
		return err
	}

	return nil
}

func (o *defaultAcquirer) Acquire() (Lock, error) {
	err := o.validateCommon()
	if err != nil {
		return nil, err
	}

	return timedCreateMutex(o.location, o.acquireTimeout)
}

func timedCreateMutex(name string, timeout time.Duration) (*windowsLock, error) {
	api, err := loadCreateMutexApi()
	if err != nil {
		return nil, err
	}

	start := time.Now()

	for time.Since(start) < timeout {
		// TODO: Global should be an OS specific option.
		mutexName := uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(globalPrefix + name)))
		mutexHandle, _, err := api.createMutexProc.Call(0, 0, mutexName)
		errNum := int(err.(windows.Errno))
		if mutexHandle == 0 || errNum > 0 {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		_, _, err = api.waitForSingleObjectProc.Call(mutexHandle, uintptr(timeout.Seconds() * 1000))
		errNum = int(err.(windows.Errno))
		if errNum > 0 {
			return nil, &AcquireError{
				reason: fmt.Sprintf("%s got return code %d - %s",
					unableToAcquirePrefix, int(errNum), err.Error()),
				inUse:  true,
			}
		}

		return &windowsLock{
			kernel32:    api.kernel32,
			mutexHandle: mutexHandle,
		}, nil
	}

	return nil, &AcquireError{
		reason: fmt.Sprintf("%s failed to acquire lock after %s",
			unableToAcquirePrefix, timeout.String()),
		inUse:  true,
	}
}

type createMutexApi struct {
	kernel32                *windows.LazyDLL
	createMutexProc         *windows.LazyProc
	waitForSingleObjectProc *windows.LazyProc
}

func loadCreateMutexApi() (*createMutexApi, error) {
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

	return &createMutexApi{
		kernel32:                kernel32,
		createMutexProc:         createMutexProc,
		waitForSingleObjectProc: waitForSingleObjectProc,
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
