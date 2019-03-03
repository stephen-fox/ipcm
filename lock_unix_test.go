// +build !windows

package lock

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

const (
	lockFileName = "junk"
)

func TestDefaultAcquirer_Acquire(t *testing.T) {
	dirPath := setupLockFileTestEnv(t)
	lockFilePath := path.Join(dirPath, lockFileName)

	l, err := NewAcquirer().
		SetLocation(lockFilePath).
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

	src := `package main

import (
	"log"

	"github.com/stephen-fox/lock"
)

func main() {
	l, err := lock.NewAcquirer().
		SetLocation("` + lockFilePath + `").
		Acquire()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer l.Release()
}
`
	_, err = prepareTestHarness([]byte(src), dirPath, t).CombinedOutput()
	if err == nil {
		t.Fatal("expected test harness lock acquire to fail, but it did not")
	}
}

func TestDefaultAcquirer_Acquire_RelativePath(t *testing.T) {
	l, err := NewAcquirer().
		SetLocation("not-fully-a-qualified-path").
		Acquire()
	if err == nil {
		l.Release()
		t.Fatal("acquisition of relative path did not fail")
	}
}

func TestDefaultAcquirer_Acquire_CustomTimeout(t *testing.T) {
	dirPath := setupLockFileTestEnv(t)
	lockFilePath := path.Join(dirPath, lockFileName)
	testHarness := newProcessAcquiresLockAndIdles(dirPath, lockFilePath, t)
	defer func() {
		err := testHarness.Process.Kill()
		if err != nil {
			t.Fatal(err.Error())
		}
	}()

	acquireTimeout := 5 * time.Second
	start := time.Now()
	l, err := NewAcquirer().
		SetLocation(lockFilePath).
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
	dirPath := setupLockFileTestEnv(t)
	lockFilePath := path.Join(dirPath, lockFileName)
	testHarness := newProcessAcquiresLockAndIdles(dirPath, lockFilePath, t)
	defer func() {
		err := testHarness.Process.Kill()
		if err != nil {
			t.Fatal(err.Error())
		}
	}()

	l, err := NewAcquirer().
		SetLocation(lockFilePath).
		Acquire()
	if err == nil {
		l.Release()
		t.Fatal("expected acquire attempt to fail")
	}
}

func newProcessAcquiresLockAndIdles(testDataDirPath string, lockFilePath string, t *testing.T) *exec.Cmd {
	src := `package main

import (
	"fmt"
	"log"
	"time"

	"github.com/stephen-fox/lock"
)

func main() {
	l, err := lock.NewAcquirer().
		SetLocation("` + lockFilePath + `").
		Acquire()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer l.Release()

	for {
		fmt.Println("ready")
		time.Sleep(1 * time.Second)
	}
}
`

	testHarness := prepareTestHarness([]byte(src), testDataDirPath, t)

	// Need to start test harness async. We need to be able to
	// test exactly when the harness acquires the lock, otherwise
	// there is a race condition between the harness and the unit
	// test when acquiring the lock.
	stdout := bytes.NewBuffer(nil)
	testHarness.Stdout = stdout
	stderr := bytes.NewBuffer(nil)
	testHarness.Stderr = stderr

	err := testHarness.Start()
	if err != nil {
		t.Fatalf("test harness lock failed - %s", err.Error())
	}

	start := time.Now()
	for {
		if testHarness.ProcessState != nil && testHarness.ProcessState.Exited() {
			t.Fatalf("test harness exited unexpectedly - output: %s", stderr.String())
		}
		if stdout.Len() > 0 {
			break
		}
		duration := time.Since(start)
		if duration >= 5 * time.Second {
			testHarness.Process.Kill()
			t.Fatalf("test harness failed to acquire lock after %s", duration.String())
		}
		time.Sleep(1 * time.Second)
	}

	return testHarness
}

func setupLockFileTestEnv(t *testing.T) (testDataDirPath string) {
	dirPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory for testing - %s", err.Error())
	}

	_, err = os.Stat(path.Join(dirPath, "go.mod"))
	if err != nil {
		t.Fatalf("current working directory is not repo - %s", err.Error())
	}

	dirPath = path.Join(dirPath, ".testdata")

	err = os.MkdirAll(dirPath, 0700)
	if err != nil {
		t.Fatalf("failed to create test data directory - %s", err.Error())
	}

	return dirPath
}

func prepareTestHarness(src []byte, dirPath string, t *testing.T) *exec.Cmd {
	srcFilePath := path.Join(dirPath, "testharness.go")

	err := ioutil.WriteFile(srcFilePath, []byte(src), 0600)
	if err != nil {
		t.Fatalf("failed to write testharness source - %s", err.Error())
	}

	testHarnessExePath := path.Join(dirPath, "testharness")

	raw, err := exec.Command("go", "build", "-o", testHarnessExePath, srcFilePath).CombinedOutput()
	if err != nil {
		t.Fatalf("test harness source failed to compile - %s - %s", err.Error(), raw)
	}

	// Note: If we decide to use 'go run (whatever.go)' instead,
	// we need to make sure its process group ID (PGID) gets set
	// to the same value as the exe that it compiles. This is
	// avoided by using 'go build (whatever)' and then executing
	// the compiled binary.
	return exec.Command(testHarnessExePath)
}
