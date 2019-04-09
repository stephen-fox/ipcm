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
	reason     string
	createFail bool
	readFail   bool
	inUse      bool
	dirFail    bool
	dllFail    bool
	procFail   bool
	syncFail   bool
}

func (o *AcquireError) Error() string {
	return o.reason
}

func (o *AcquireError) FailedToCreated() bool {
	return o.createFail
}

func (o *AcquireError) ReadFailed() bool {
	return o.readFail
}

func (o *AcquireError) AnotherInstanceOwnsLock() bool {
	return o.inUse
}

func (o *AcquireError) FailedToCreateParentDirectory() bool {
	return o.dirFail
}

func (o *AcquireError) WindowsDllLoadFailed() bool {
	return o.dllFail
}

func (o *AcquireError) WindowsProcedureFailed() bool {
	return o.procFail
}

func (o *AcquireError) SyncMutexLockTimedOut() bool {
	return o.syncFail
}
