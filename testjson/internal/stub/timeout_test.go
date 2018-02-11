// +build stubpkg,timeoutstub

package stub

import (
	"testing"
	"time"
)

func TestTimeout(t *testing.T) {
	time.Sleep(time.Minute)
}
