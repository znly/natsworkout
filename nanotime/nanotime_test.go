package nanotime

import (
	"testing"
	"time"
)

func TestNanoTimeNow(t *testing.T) {
	for i := 0; i < 1000; i++ {
		nt := Now()
		tt := time.Unix(0, int64(nt))
		t.Logf("nanotime=%d time=%s", nt, tt)
	}
}
