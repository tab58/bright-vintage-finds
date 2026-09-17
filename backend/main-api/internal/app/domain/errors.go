// Package domain holds the inventory's business types and rules. It imports
// nothing but the standard library: no Ent, no huma, no net/http. Anything
// that needs a database row or an HTTP status code belongs in an adapter.
package domain

import (
	"errors"
	"fmt"
)

// Kind classifies a domain error so the driving adapter can pick a status
// code without inspecting messages.
type Kind uint8

const (
	// KindInternal is an unexpected failure; the adapter reports it as a 500
	// and the message is not meant for the caller.
	KindInternal Kind = iota
	// KindInvalid is a caller mistake in the payload.
	KindInvalid
	// KindNotFound is a missing or soft-deleted record.
	KindNotFound
	// KindConflict is a request that collides with existing state.
	KindConflict
	// KindTooLarge is a payload over a size cap.
	KindTooLarge
	// KindUnavailable is a dependency the deployment has not configured.
	KindUnavailable
)

// Error is a domain failure carrying the message the caller sees. The message
// is part of the API contract, so it is written here rather than at the edge.
type Error struct {
	Kind Kind
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *Error) Unwrap() error { return e.Err }

// Is matches on kind alone, so callers can test errors.Is(err, domain.ErrNotFound).
func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	return ok && other.Msg == "" && other.Err == nil && other.Kind == e.Kind
}

// Kind sentinels for errors.Is checks.
var (
	ErrInvalid     = &Error{Kind: KindInvalid}
	ErrNotFound    = &Error{Kind: KindNotFound}
	ErrConflict    = &Error{Kind: KindConflict}
	ErrTooLarge    = &Error{Kind: KindTooLarge}
	ErrUnavailable = &Error{Kind: KindUnavailable}
)

func Invalid(msg string) error                { return &Error{Kind: KindInvalid, Msg: msg} }
func Invalidf(format string, a ...any) error  { return Invalid(fmt.Sprintf(format, a...)) }
func NotFound(msg string) error               { return &Error{Kind: KindNotFound, Msg: msg} }
func Conflict(msg string) error               { return &Error{Kind: KindConflict, Msg: msg} }
func Conflictf(format string, a ...any) error { return Conflict(fmt.Sprintf(format, a...)) }
func TooLarge(msg string) error               { return &Error{Kind: KindTooLarge, Msg: msg} }
func Unavailable(msg string) error            { return &Error{Kind: KindUnavailable, Msg: msg} }
func Internal(msg string, err error) error    { return &Error{Kind: KindInternal, Msg: msg, Err: err} }

// KindOf reports the kind of err, defaulting to KindInternal for anything
// that did not come from this package.
func KindOf(err error) Kind {
	var de *Error
	if errors.As(err, &de) {
		return de.Kind
	}
	return KindInternal
}
