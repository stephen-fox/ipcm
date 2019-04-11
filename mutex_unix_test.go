// +build !windows

package lock

import (
	"testing"
)

func TestNewMutex_RelativePath(t *testing.T) {
	_, err := NewMutex(MutexConfig{
		Resource: "no-a-fully-qualified-path",
	})
	if err == nil {
		t.Fatal("acquisition of relative path did not fail")
	}
}
