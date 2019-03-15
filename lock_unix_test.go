// +build !windows

package lock

import (
	"path"
	"testing"
	"time"
)

const (
	lockFileName = "junk"
)

func TestDefaultAcquirer_Acquire(t *testing.T) {
	env := setupLockFileTestEnv(t)
	lockFilePath := path.Join(env.dataDirPath, lockFileName)

	l, err := NewAcquirer().
		SetResource(lockFilePath).
		Acquire()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer func() {
		err := l.Release()
		if err != nil {
			t.Fatal(err.Error())
		}
	}()

	o := testHarnessOptions{
		resource: lockFilePath,
	}

	_, err = prepareTestHarness(env, o, t).CombinedOutput()
	if err == nil {
		t.Fatal("expected test harness lock acquire to fail, but it did not")
	}
}

func TestDefaultAcquirer_Acquire_RelativePath(t *testing.T) {
	l, err := NewAcquirer().
		SetResource("not-fully-a-qualified-path").
		Acquire()
	if err == nil {
		l.Release()
		t.Fatal("acquisition of relative path did not fail")
	}
}

func TestDefaultAcquirer_Acquire_CustomTimeout(t *testing.T) {
	env := setupLockFileTestEnv(t)
	lockFilePath := path.Join(env.dataDirPath, lockFileName)
	testHarness := newProcessAcquiresLockAndIdles(env, lockFilePath, t)
	defer func() {
		err := testHarness.Process.Kill()
		if err != nil {
			t.Fatal(err.Error())
		}
	}()

	acquireTimeout := 5 * time.Second
	start := time.Now()
	l, err := NewAcquirer().
		SetResource(lockFilePath).
		SetAcquireTimeout(acquireTimeout).
		Acquire()
	if err == nil {
		l.Release()
		t.Fatal("expected acquire attempt to fail")
	}

	duration := time.Since(start)
	if duration < acquireTimeout {
		t.Fatalf("timeout only lasted %s when it should have taken at least %s",
			duration.String(), acquireTimeout.String())
	}
}

func TestDefaultAcquirer_Acquire_AlreadyAcquired(t *testing.T) {
	env := setupLockFileTestEnv(t)
	lockFilePath := path.Join(env.dataDirPath, lockFileName)
	testHarness := newProcessAcquiresLockAndIdles(env, lockFilePath, t)
	defer func() {
		err := testHarness.Process.Kill()
		if err != nil {
			t.Fatal(err.Error())
		}
	}()

	l, err := NewAcquirer().
		SetResource(lockFilePath).
		Acquire()
	if err == nil {
		l.Release()
		t.Fatal("expected acquire attempt to fail")
	}
}
