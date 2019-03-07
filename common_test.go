package lock

import (
	"bytes"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

type testEnv struct {
	dataDirPath    string
	harnessSrcPath string
}

type testHarnessOptions struct {
	lockLocation string
	loopForever  bool
}

func (o testHarnessOptions) args(t *testing.T) []string {
	if len(o.lockLocation) == 0 {
		t.Fatal("lock location was not specified for test harness")
	}

	args := []string{"-location", o.lockLocation}

	if o.loopForever {
		args = append(args, "-loop")
	}

	return args
}

func setupLockFileTestEnv(t *testing.T) testEnv {
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

	return testEnv{
		dataDirPath:    testDataDir,
		harnessSrcPath: path.Join(dirPath, "cmd/testharness/main.go"),
	}
}

func prepareTestHarness(env testEnv, options testHarnessOptions, t *testing.T) *exec.Cmd {
	testHarnessExePath := path.Join(env.dataDirPath, "testharness")

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

func newProcessAcquiresLockAndIdles(env testEnv, lockLocation string, t *testing.T) *exec.Cmd {
	o := testHarnessOptions{
		lockLocation: lockLocation,
		loopForever:  true,
	}
	testHarness := prepareTestHarness(env, o, t)

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

