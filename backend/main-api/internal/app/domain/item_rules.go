package domain

import "time"

// StatusStamp is the listing-timestamp bookkeeping a status move implies.
type StatusStamp struct {
	ListedAt      *time.Time
	FirstListedAt *time.Time
	ClearListedAt bool
}

// ItemStatusState is the part of a stored item a status change reads.
type ItemStatusState struct {
	Status        Status
	HasSoldPlace  bool
	FirstListedAt *time.Time
}

// PlanCreateStatus decides the timestamps for an item created directly in a
// given status. An item created as listed is listed for the first time, so
// both timestamps are stamped.
func PlanCreateStatus(status Status, hasSoldPlace bool, now time.Time) (StatusStamp, error) {
	if status == StatusSold && !hasSoldPlace {
		return StatusStamp{}, Invalid("a sold item needs sold_place_id")
	}
	if status == StatusListed {
		t := now
		return StatusStamp{ListedAt: &t, FirstListedAt: &t}, nil
	}
	return StatusStamp{}, nil
}

// PlanStatusChange decides the timestamps for moving a stored item to next.
// Re-sending the current status is a no-op on the timestamps, so a listed
// item does not have its listing date reset by an unrelated edit.
func PlanStatusChange(next Status, current ItemStatusState, patchSetsSoldPlace bool, now time.Time) (StatusStamp, error) {
	// A sold item always names where it sold: sales insight is built on that
	// edge. The place comes from this payload or is already stored.
	if next == StatusSold && !patchSetsSoldPlace && !current.HasSoldPlace {
		return StatusStamp{}, Invalid("a sold item needs sold_place_id")
	}

	switch {
	case next == StatusListed && current.Status != StatusListed:
		t := now
		stamp := StatusStamp{ListedAt: &t}
		// first_listed_at is written once and then left alone.
		if current.FirstListedAt == nil {
			stamp.FirstListedAt = &t
		}
		return stamp, nil
	case next == StatusDraft:
		return StatusStamp{ClearListedAt: true}, nil
	default:
		return StatusStamp{}, nil
	}
}

// CanDelete reports whether an item may be deleted. Deletable: a draft that
// never went out, or anything already archived — archiving is the deliberate
// step before disposal. A live listing must be unlisted or archived first,
// and a sale is never deleted.
func CanDelete(status Status, firstListedAt *time.Time) error {
	neverListed := status == StatusDraft && firstListedAt == nil
	switch {
	case status == StatusArchived || neverListed:
		return nil
	case status == StatusSold:
		return Conflict("sold items are a record of the sale and cannot be deleted")
	default:
		return Conflict("this item has been listed; archive it before deleting")
	}
}
