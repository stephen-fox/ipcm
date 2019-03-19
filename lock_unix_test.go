// +build !windows

package lock

import (
	"io/ioutil"
	"path"
	"strconv"
	"sync"
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

	testHarness.Process.Kill()
	testHarness.Wait()

	err = m.TryLock()
	if err != nil {
		t.Fatalf("lock attempt failed after killing holder - %s", err.Error())
	}
	m.Unlock()
}

func TestNewMutex_MultipleRoutines(t *testing.T) {
	env := setupLockFileTestEnv(t)
	lockFilePath := path.Join(env.dataDirPath, lockFileName)
	m, err := NewMutex(lockFilePath)
	if err != nil {
		t.Fatal(err.Error())
	}

	const exp = 100
	result := 0
	wg := &sync.WaitGroup{}
	wg.Add(exp)

	for i := 0; i < exp; i++ {
		go func() {
			m.Lock()
			result++
			m.Unlock()
			wg.Done()
		}()
	}

	wg.Wait()

	if result != exp {
		t.Fatal("got", result, "- expected", exp)
	}
}

func TestNewMutex_MultipleRoutinesIpc(t *testing.T) {
	env := setupLockFileTestEnv(t)

	ipcFilePath := path.Join(env.dataDirPath, "whatever.txt")
	err := ioutil.WriteFile(ipcFilePath, []byte{'0'}, 0600)
	if err != nil {
		t.Fatal(err.Error())
	}

	const expected = 100
	const half = expected / 2

	lockFilePath := path.Join(env.dataDirPath, lockFileName)
	options := testHarnessOptions{
		resource:    lockFilePath,
		ipcFilePath: ipcFilePath,
		ipcValue:    half,
	}

	testHarness := prepareTestHarness(env, options, t)

	m, err := NewMutex(lockFilePath)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testHarness.Start()
	if err != nil {
		t.Fatalf("failed to start test hanress - %s", err.Error())
	}
	defer func() {
		testHarness.Process.Kill()
		testHarness.Wait()
	}()

	wg := &sync.WaitGroup{}
	wg.Add(half)

	for i := 0; i < half; i++ {
		go func() {
			m.Lock()
			defer m.Unlock()

			raw, err := ioutil.ReadFile(ipcFilePath)
			if err != nil {
				t.Fatalf("failed to read IPC test file - %s", err.Error())
			}

			v, err := strconv.Atoi(string(raw))
			if err != nil {
				t.Fatalf("failed to read an integer from IPC test file - %s", err.Error())
			}

			v++
			err = ioutil.WriteFile(ipcFilePath, []byte(strconv.Itoa(v)), 0600)
			if err != nil {
				t.Fatalf("failed to write to IPC test file - %s", err.Error())
			}

			wg.Done()
		}()
	}

	err = testHarness.Wait()
	if err != nil {
		t.Fatalf("failed to wait for test harness - %s", err.Error())
	}

	wg.Wait()

	raw, err := ioutil.ReadFile(ipcFilePath)
	if err != nil {
		t.Fatalf("failed to read final value from IPC test file - %s", err.Error())
	}

	final, err := strconv.Atoi(string(raw))
	if err != nil {
		t.Fatalf("failed to convert final value - %s", err.Error())
	}

	if final != expected {
		t.Fatalf("final value in IPC test file should be %d - got %d", expected, final)
	}
}
