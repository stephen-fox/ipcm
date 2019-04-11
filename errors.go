package lock

type ConfigureError struct {
	reason     string
	noResource bool
	notAbs     bool
}

func (o *ConfigureError) Error() string {
	return o.reason
}

func (o *ConfigureError) ResourceNotSpecified() bool {
	return o.noResource
}

func (o *ConfigureError) PathNotFullyQualified() bool {
	return o.notAbs
}

type LockError struct {
	reason        string
	createFail    bool
	dirFail       bool
	dllLoadFail   bool
	procLoadFail  bool
	syncTimeout   bool
	systemTimeout bool
	syscallFailed bool
}

func (o *LockError) Error() string {
	return o.reason
}

func (o *LockError) FailedToCreated() bool {
	return o.createFail
}

func (o *LockError) FailedToCreateParentDirectory() bool {
	return o.dirFail
}

func (o *LockError) WindowsDllLoadFailed() bool {
	return o.dllLoadFail
}

func (o *LockError) WindowsProcedureLoadFailed() bool {
	return o.procLoadFail
}

func (o *LockError) SyncMutexLockTimedOut() bool {
	return o.syncTimeout
}

func (o *LockError) SystemMutexLockTimedOut() bool {
	return o.systemTimeout
}

func (o *LockError) SystemCallFailed() bool {
	return o.syscallFailed
}
