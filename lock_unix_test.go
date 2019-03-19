// +build !windows

package lock

import (
	"log"
	"path"
	"testing"
	"time"
)

const (
	lockFileName = "junk"
)

func TestNewMutex(t *testing.T) {
	env := setupLockFileTestEnv(t)
	lockFilePath := path.Join(env.dataDirPath, lockFileName)

	m, err := NewMutex(lockFilePath)
	if err != nil {
		t.Fatal(err.Error())
	}

	m.Lock()
	defer m.Unlock()

	o := testHarnessOptions{
		resource: lockFilePath,
		once:     true,
	}

	_, err = prepareTestHarness(env, o, t).CombinedOutput()
	if err == nil {
		t.Fatal("expected test harness lock acquire to fail, but it did not")
	}
}

func TestNewMutex_RelativePath(t *testing.T) {
	_, err := NewMutex("not-fully-a-qualified-path")
	if err == nil {
		t.Fatal("acquisition of relative path did not fail")
	}
}

func TestNewMutex_TimedTryLock(t *testing.T) {
	env := setupLockFileTestEnv(t)
	lockFilePath := path.Join(env.dataDirPath, lockFileName)
	testHarness := newProcessAcquiresLockAndIdles(env, lockFilePath, t)
	defer func() {
		testHarness.Process.Kill()
		testHarness.Wait()
	}()

	m, err := NewMutex(lockFilePath)
	if err != nil {
		t.Fatal(err.Error())
	}

	start := time.Now()
	acquireTimeout := 5 * time.Second
	err = m.TimedTryLock(acquireTimeout)
	if err == nil {
		t.Fatal("expected acquire attempt to fail")
	}

	duration := time.Since(start)
	if duration < acquireTimeout {
		t.Fatalf("timeout only lasted %s when it should have taken at least %s",
			duration.String(), acquireTimeout.String())
	}

	testHarness.Process.Kill()
	testHarness.Wait()

	err = m.TimedTryLock(acquireTimeout)
	if err != nil {
		t.Fatalf("try lock should have succeeded, but it failed - %s", err.Error())
	}
	m.Unlock()
}

func TestNewMutex_TryLock(t *testing.T) {
	env := setupLockFileTestEnv(t)
	lockFilePath := path.Join(env.dataDirPath, lockFileName)
	testHarness := newProcessAcquiresLockAndIdles(env, lockFilePath, t)
	defer func() {
		testHarness.Process.Kill()
		testHarness.Wait()
	}()

	m, err := NewMutex(lockFilePath)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = m.TryLock()
	if err == nil {
		t.Fatal("expected acquire attempt to fail")
	}
}
