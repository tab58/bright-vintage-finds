package ports

import "time"

// Clock reads the current time. Services take one so a test can pin "now"
// without the indirection of an interface.
type Clock func() time.Time

// SystemClock is the real clock, in UTC — every timestamp this service writes
// is UTC.
func SystemClock() time.Time { return time.Now().UTC() }

// IDGenerator returns a fresh opaque id, used for object keys. The database
// mints its own row ids; this is only for names the application chooses.
type IDGenerator func() string
