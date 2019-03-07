package lock

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	kernel32Name = "kernel32.dll"
	createMutexW = "CreateMutexW"
	globalPrefix = "Global\\"
)

type windowsLock struct {

}

func (o *windowsLock) Release() error {
	// TODO:
	return nil
}

func (o *defaultAcquirer) Acquire() (Lock, error) {
	err := o.validateCommon()
	if err != nil {
		return nil, err
	}

	k32 := windows.NewLazyDLL(kernel32Name)
	if k32 == nil {
		return nil, &AcquireError{
			reason:  fmt.Sprintf("%s failed to load %s",
				unableToCreatePrefix, kernel32Name),
			dllFail: true,
		}
	}

	procCreateMutex := k32.NewProc(createMutexW)
	if procCreateMutex == nil {
		return nil, &AcquireError{
			reason:   fmt.Sprintf("%s failed to load %s procedure from %s",
				unableToCreatePrefix, createMutexW, kernel32Name),
			procFail: true,
		}
	}

	// TODO: Add timeout support.

	// TODO: Global should be an OS specific option.
	ptr := uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(globalPrefix + o.location)))
	// TODO: Save the returned pointer and release it?
	_, _, err = procCreateMutex.Call(0, 0, ptr)
	errNum := int(err.(windows.Errno))
	if errNum > 0 {
		return nil, &AcquireError{
			reason: fmt.Sprintf("%s got return code %d - %s",
				unableToAcquirePrefix, int(err.(windows.Errno)), err.Error()),
			inUse:  true,
		}
	}

	return &windowsLock{

	}, nil
}
