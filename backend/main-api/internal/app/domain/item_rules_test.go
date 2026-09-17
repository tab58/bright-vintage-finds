package domain

import (
	"testing"
	"time"
)

var now = time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

func TestPlanCreateStatus(t *testing.T) {
	tests := []struct {
		name         string
		status       Status
		hasSoldPlace bool
		wantErr      string
		wantListed   bool
		wantFirst    bool
	}{
		{name: "draft stamps nothing", status: StatusDraft},
		{name: "archived stamps nothing", status: StatusArchived},
		{name: "listed stamps both timestamps", status: StatusListed, wantListed: true, wantFirst: true},
		{name: "sold without a place is rejected", status: StatusSold, wantErr: "a sold item needs sold_place_id"},
		{name: "sold with a place is allowed", status: StatusSold, hasSoldPlace: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PlanCreateStatus(tt.status, tt.hasSoldPlace, now)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("want error, got nil")
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("message = %q, want %q", err.Error(), tt.wantErr)
				}
				if KindOf(err) != KindInvalid {
					t.Errorf("kind = %d, want KindInvalid", KindOf(err))
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if (got.ListedAt != nil) != tt.wantListed {
				t.Errorf("ListedAt set = %v, want %v", got.ListedAt != nil, tt.wantListed)
			}
			if (got.FirstListedAt != nil) != tt.wantFirst {
				t.Errorf("FirstListedAt set = %v, want %v", got.FirstListedAt != nil, tt.wantFirst)
			}
			if got.ClearListedAt {
				t.Error("creation never clears listed_at")
			}
		})
	}
}

func TestPlanStatusChange(t *testing.T) {
	earlier := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		next          Status
		current       ItemStatusState
		patchHasPlace bool
		wantErr       string
		wantListedAt  bool
		wantFirstAt   bool
		wantClear     bool
	}{
		{
			name:         "draft to listed stamps both on a first listing",
			next:         StatusListed,
			current:      ItemStatusState{Status: StatusDraft},
			wantListedAt: true,
			wantFirstAt:  true,
		},
		{
			name:         "relisting refreshes listed_at but keeps first_listed_at",
			next:         StatusListed,
			current:      ItemStatusState{Status: StatusDraft, FirstListedAt: &earlier},
			wantListedAt: true,
			wantFirstAt:  false,
		},
		{
			name:    "re-sending listed leaves the timestamps alone",
			next:    StatusListed,
			current: ItemStatusState{Status: StatusListed, FirstListedAt: &earlier},
		},
		{
			name:      "back to draft clears listed_at",
			next:      StatusDraft,
			current:   ItemStatusState{Status: StatusListed, FirstListedAt: &earlier},
			wantClear: true,
		},
		{
			name:    "archiving touches no timestamps",
			next:    StatusArchived,
			current: ItemStatusState{Status: StatusListed, FirstListedAt: &earlier},
		},
		{
			name:    "sold with no place anywhere is rejected",
			next:    StatusSold,
			current: ItemStatusState{Status: StatusListed},
			wantErr: "a sold item needs sold_place_id",
		},
		{
			name:          "sold with a place in the payload is allowed",
			next:          StatusSold,
			current:       ItemStatusState{Status: StatusListed},
			patchHasPlace: true,
		},
		{
			name:    "sold with a place already stored is allowed",
			next:    StatusSold,
			current: ItemStatusState{Status: StatusSold, HasSoldPlace: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PlanStatusChange(tt.next, tt.current, tt.patchHasPlace, now)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if (got.ListedAt != nil) != tt.wantListedAt {
				t.Errorf("ListedAt set = %v, want %v", got.ListedAt != nil, tt.wantListedAt)
			}
			if (got.FirstListedAt != nil) != tt.wantFirstAt {
				t.Errorf("FirstListedAt set = %v, want %v", got.FirstListedAt != nil, tt.wantFirstAt)
			}
			if got.ClearListedAt != tt.wantClear {
				t.Errorf("ClearListedAt = %v, want %v", got.ClearListedAt, tt.wantClear)
			}
			if got.ListedAt != nil && !got.ListedAt.Equal(now) {
				t.Errorf("ListedAt = %v, want %v", got.ListedAt, now)
			}
		})
	}
}

func TestCanDelete(t *testing.T) {
	listedOnce := time.Date(2025, 5, 5, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		status        Status
		firstListedAt *time.Time
		wantErr       string
	}{
		{name: "never-listed draft is deletable", status: StatusDraft},
		{name: "archived is deletable", status: StatusArchived, firstListedAt: &listedOnce},
		{
			name:          "draft that was once listed is not",
			status:        StatusDraft,
			firstListedAt: &listedOnce,
			wantErr:       "this item has been listed; archive it before deleting",
		},
		{
			name:    "live listing is not",
			status:  StatusListed,
			wantErr: "this item has been listed; archive it before deleting",
		},
		{
			name:    "sold is never deletable",
			status:  StatusSold,
			wantErr: "sold items are a record of the sale and cannot be deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CanDelete(tt.status, tt.firstListedAt)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("message = %q, want %q", err.Error(), tt.wantErr)
			}
			if KindOf(err) != KindConflict {
				t.Errorf("kind = %d, want KindConflict", KindOf(err))
			}
		})
	}
}
