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

type AcquireError struct {
	reason        string
	createFail    bool
	dirFail       bool
	dllLoadFail   bool
	procLoadFail  bool
	syncTimeout   bool
	systemTimeout bool
	syscallFailed bool
}

func (o *AcquireError) Error() string {
	return o.reason
}

func (o *AcquireError) FailedToCreated() bool {
	return o.createFail
}

func (o *AcquireError) FailedToCreateParentDirectory() bool {
	return o.dirFail
}

func (o *AcquireError) WindowsDllLoadFailed() bool {
	return o.dllLoadFail
}

func (o *AcquireError) WindowsProcedureLoadFailed() bool {
	return o.procLoadFail
}

func (o *AcquireError) SyncMutexLockTimedOut() bool {
	return o.syncTimeout
}

func (o *AcquireError) SystemMutexLockTimedOut() bool {
	return o.systemTimeout
}

func (o *AcquireError) SystemCallFailed() bool {
	return o.syscallFailed
}
