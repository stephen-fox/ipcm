// +build !windows

package lock

import (
	"testing"
)

func TestNewMutex_RelativePath(t *testing.T) {
	_, err := NewMutex("not-fully-a-qualified-path")
	if err == nil {
		t.Fatal("acquisition of relative path did not fail")
	}
}
