// +build windows

package lock

import (
	"math/rand"
	"testing"
	"time"
)

func TestDefaultAcquirer_Acquire(t *testing.T) {
	env := setupLockFileTestEnv(t)
	lockName := randomAlphaString(10)

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
	}

	_, err = prepareTestHarness(env, o, t).CombinedOutput()
	if err == nil {
		t.Fatal("expected test harness lock acquire to fail, but it did not")
	}
}

func TestDefaultAcquirer_Acquire_CustomTimeout(t *testing.T) {
	env := setupLockFileTestEnv(t)
	lockName := randomAlphaString(10)
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
	env := setupLockFileTestEnv(t)
	lockName := randomAlphaString(10)
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

// randomAlphaString by "icza":
// https://stackoverflow.com/a/31832326
func randomAlphaString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63() % int64(len(letterBytes))]
	}

	return string(b)
}
