package nanotime

import (
	"time"
	_ "unsafe" // since we use go:linkname
)

//go:noescape
//go:linkname nanotime runtime.nanotime
func nanotime() int64

type wallClock struct {
	t      time.Time
	offset time.Duration
}

var wc wallClock

func init() {
	wc = wallClock{t: time.Now(), offset: Offset()}
}

// Now returns the current wall clock time.
func Now() time.Time {
	return wc.t.Add(Offset() - wc.offset)
}

// Offset returns a nanotime offset from an arbitrary point in time.
func Offset() time.Duration {
	return time.Duration(nanotime())
}
