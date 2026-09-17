// Package fakes holds in-memory implementations of the repository and storage
// ports, for testing services without Postgres or S3.
//
// They are as faithful as the service tests need and no more: query support
// covers the filters the application actually issues, not SQL in general. A
// fake reproduces the rules the real adapter is responsible for — unique
// whatnot numbers, soft-delete visibility, display ordering — so a service
// test exercising those paths means something.
//
// Every fake carries an Err field. When set, each method returns it untouched,
// which is how a test drives the error paths.
package fakes

import (
	"strconv"
	"time"

	"main-api/internal/app/ports"
)

// FixedTime is the instant the fake clock reports.
var FixedTime = time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

// Clock returns a ports.Clock pinned to FixedTime.
func Clock() ports.Clock { return func() time.Time { return FixedTime } }

// IDs returns a ports.IDGenerator handing out prefix1, prefix2, and so on, so
// generated object keys are predictable in assertions.
func IDs(prefix string) ports.IDGenerator {
	n := 0
	return func() string {
		n++
		return prefix + strconv.Itoa(n)
	}
}

// seq mints sequential row ids. KSUIDs sort by creation and the catalog's
// cursor relies on that, so the fake ids are zero-padded to sort the same way.
type seq struct {
	prefix string
	n      int
}

func (s *seq) next() string {
	s.n++
	return s.prefix + strconv.Itoa(s.n+100000)
}
