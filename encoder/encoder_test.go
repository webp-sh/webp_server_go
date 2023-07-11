package encoder

import (
	"sync"
	"testing"
)

var (
	VipsStartOnce sync.Once
)

func VipsSetupForTests(t *testing.T) {
	VipsStartOnce.Do(func() { VipsStart() })
}
