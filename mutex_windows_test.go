// +build windows

package lock

import (
	"testing"
	"time"
)

func TestDefaultAcquirer_Acquire(t *testing.T) {
	env := setupTestEnv(t)
	lockName := resourceName()

	l, err := NewAcquirer().
		SetResource(lockName).
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
		resource: lockName,
		once:     true,
	}

	_, err = prepareTestHarness(env, o, t).CombinedOutput()
	if err == nil {
		t.Fatal("expected test harness lock acquire to fail, but it did not")
	}
}

func TestDefaultAcquirer_Acquire_CustomTimeout(t *testing.T) {
	env := setupTestEnv(t)
	lockName := resourceName()
	testHarness := newProcessAcquiresLockAndIdles(env, lockName, t)
	defer func() {
		err := testHarness.Process.Kill()
		if err != nil {
			t.Fatal(err.Error())
		}
	}()

	acquireTimeout := 5 * time.Second
	start := time.Now()
	l, err := NewAcquirer().
		SetResource(lockName).
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
	env := setupTestEnv(t)
	lockName := resourceName()
	testHarness := newProcessAcquiresLockAndIdles(env, lockName, t)
	defer func() {
		err := testHarness.Process.Kill()
		if err != nil {
			t.Fatal(err.Error())
		}
	}()

	l, err := NewAcquirer().
		SetResource(lockName).
		Acquire()
	if err == nil {
		l.Release()
		t.Fatal("expected acquire attempt to fail")
	}
}
