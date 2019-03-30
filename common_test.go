package lock

import (
	"bytes"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// testEnv contains information about the test environment.
type testEnv struct {
	resource       string
	dataDirPath    string
	harnessSrcPath string
}

// testHarnessOptions represents all of the possible options available when
// running the test harness.
type testHarnessOptions struct {
	// resource is the external resource to manipulate (e.g., a
	// fle path).
	resource    string
	once        bool
	loopForever bool
	ipcFilePath string
	ipcValue    int
}

func (o testHarnessOptions) args(t *testing.T) []string {
	if len(o.resource) == 0 {
		t.Fatal("lock resource was not specified for test harness")
	}

	args := []string{"-resource", o.resource}

	if o.once {
		args = append(args, "-once")
	}

	if o.loopForever {
		args = append(args, "-loop")
	}

	if len(o.ipcFilePath) > 0 {
		args = append(args, "-ipcfile", o.ipcFilePath)
		args = append(args, "-ipcvalue", strconv.Itoa(o.ipcValue))
	}

	return args
}

// setupTestEnv creates the test data directory and gets information about
// the repository.
func setupTestEnv(t *testing.T) testEnv {
	dirPath, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory for testing - %s", err.Error())
	}

	_, err = os.Stat(path.Join(dirPath, "go.mod"))
	if err != nil {
		t.Fatalf("current working directory is not repo - %s", err.Error())
	}

	testDataDir := path.Join(dirPath, ".testdata")

	err = os.MkdirAll(testDataDir, 0700)
	if err != nil {
		t.Fatalf("failed to create test data directory - %s", err.Error())
	}

	resource := path.Join(testDataDir, "junk")
	if runtime.GOOS == "windows" {
		resource = randStringBytesRmndr(10)
	}

	return testEnv{
		resource:       resource,
		dataDirPath:    testDataDir,
		harnessSrcPath: path.Join(dirPath, "cmd/testharness/main.go"),
	}
}

// compileTestHarness compiles the test harness application and returns
// an *exec.Cmd representing the test harness with the provided
// testHarnessOptions. The returned Cmd must be started by the caller.
func compileTestHarness(env testEnv, options testHarnessOptions, t *testing.T) *exec.Cmd {
	testHarnessExePath := path.Join(env.dataDirPath, "testharness")
	if runtime.GOOS == "windows" {
		testHarnessExePath = testHarnessExePath + ".exe"
	}

	// Note: If we decide to use 'go run (whatever.go)' instead,
	// we need to make sure its process group ID (PGID) gets set
	// to the same value as the exe that it compiles. This is
	// avoided by using 'go build (whatever)' and then executing
	// the compiled binary.
	raw, err := exec.Command("go", "build", "-o", testHarnessExePath, env.harnessSrcPath).CombinedOutput()
	if err != nil {
		t.Fatalf("test harness source failed to compile - %s - %s", err.Error(), raw)
	}

	return exec.Command(testHarnessExePath, options.args(t)...)
}

// newProcessLocksAndIdles compiles and then starts the test harness.
// The test harness will acquire the mutex and then idle forever.
//
// Callers are responsible for the lifecycle of the test harness.
func newProcessLocksAndIdles(env testEnv, t *testing.T) *exec.Cmd {
	o := testHarnessOptions{
		resource:    env.resource,
		loopForever: true,
	}
	testHarness := compileTestHarness(env, o, t)

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
			t.Fatalf("test harness failed to acquire lock after %s - output: %s",
				duration.String(), stderr.String())
		}
		time.Sleep(1 * time.Second)
	}

	return testHarness
}

// randStringBytesRmndr by "icza":
// https://stackoverflow.com/a/31832326
func randStringBytesRmndr(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63() % int64(len(letterBytes))]
	}

	return string(b)
}
