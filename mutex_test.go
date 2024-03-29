package ipcm

import (
	"bytes"
	"io/ioutil"
	"path"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestNewMutex(t *testing.T) {
	env := setupTestEnv(t)

	m, err := NewMutex(env.mutexConfig)
	if err != nil {
		t.Fatal(err.Error())
	}

	m.Lock()
	defer m.Unlock()

	o := testHarnessOptions{
		config: env.mutexConfig,
	}

	_, err = compileTestHarness(env, o, t).CombinedOutput()
	if err == nil {
		t.Fatal("test harness lock should have failed")
	}
}

func TestNewMutex_TimedTryLock(t *testing.T) {
	env := setupTestEnv(t)
	testHarness := newProcessLocksAndIdles(env, t)
	defer func() {
		testHarness.Process.Kill()
		testHarness.Wait()
	}()

	m, err := NewMutex(env.mutexConfig)
	if err != nil {
		t.Fatal(err.Error())
	}

	start := time.Now()
	lockTimeout := 5 * time.Second
	err = m.TimedTryLock(lockTimeout)
	if err == nil {
		t.Fatal("lock attempt should have failed")
	}

	duration := time.Since(start)
	if duration < lockTimeout {
		t.Fatalf("lock timeout only lasted %s when it should have taken at least %s",
			duration.String(), lockTimeout.String())
	}

	testHarness.Process.Kill()
	testHarness.Wait()

	err = m.TimedTryLock(lockTimeout)
	if err != nil {
		t.Fatalf("lock should have succeeded, but it failed - %s", err.Error())
	}
	m.Unlock()
}

func TestNewMutex_MultipleRoutines(t *testing.T) {
	env := setupTestEnv(t)
	m, err := NewMutex(env.mutexConfig)
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
	env := setupTestEnv(t)

	ipcFilePath := path.Join(env.dataDirPath, "ipc-test.txt")
	err := ioutil.WriteFile(ipcFilePath, []byte{'0'}, 0600)
	if err != nil {
		t.Fatal(err.Error())
	}

	const expected = 100
	const half = expected / 2

	options := testHarnessOptions{
		config:      env.mutexConfig,
		ipcFilePath: ipcFilePath,
		ipcValue:    half,
	}

	testHarness := compileTestHarness(env, options, t)
	stderr := bytes.NewBuffer(nil)
	testHarness.Stderr = stderr

	m, err := NewMutex(env.mutexConfig)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testHarness.Start()
	if err != nil {
		t.Fatalf("failed to start test hanress - %s - output: '%s'",
			err.Error(), stderr.String())
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
			defer wg.Done()

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
		}()
	}

	err = testHarness.Wait()
	if err != nil {
		t.Fatalf("failed to wait for test harness - %s - output: '%s'",
			err.Error(), stderr.String())
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
