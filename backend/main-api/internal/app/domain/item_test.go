package domain

import (
	"testing"
	"time"
)

func TestParseStatus(t *testing.T) {
	tests := []struct {
		in      string
		want    Status
		wantErr bool
	}{
		{in: "draft", want: StatusDraft},
		{in: "listed", want: StatusListed},
		{in: "sold", want: StatusSold},
		{in: "archived", want: StatusArchived},
		{in: "Draft", wantErr: true},
		{in: "", wantErr: true},
		{in: "deleted", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseStatus(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseStatus(%q) = %q, want error", tt.in, got)
				}
				if KindOf(err) != KindInvalid {
					t.Errorf("kind = %d, want KindInvalid", KindOf(err))
				}
				want := `invalid status "` + tt.in + `"`
				if err.Error() != want {
					t.Errorf("message = %q, want %q", err.Error(), want)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseStatus(%q): %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ParseStatus(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseMeasurementUnit(t *testing.T) {
	tests := []struct {
		in      string
		want    MeasurementUnit
		wantErr bool
	}{
		{in: "inch", want: UnitInch},
		{in: "cm", want: UnitCm},
		{in: "CM", wantErr: true},
		{in: "", wantErr: true},
		{in: "mm", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseMeasurementUnit(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseMeasurementUnit(%q) = %q, want error", tt.in, got)
				}
				if err.Error() != "measurement_unit must be inch or cm" {
					t.Errorf("message = %q", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMeasurementUnit(%q): %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEffectiveFirstListedAt(t *testing.T) {
	first := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	listed := time.Date(2026, 6, 7, 8, 9, 10, 0, time.UTC)

	tests := []struct {
		name string
		item Item
		want *time.Time
	}{
		{"both set prefers first_listed_at", Item{FirstListedAt: &first, ListedAt: &listed}, &first},
		{"legacy row falls back to listed_at", Item{ListedAt: &listed}, &listed},
		{"never listed", Item{}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.EffectiveFirstListedAt()
			switch {
			case tt.want == nil && got != nil:
				t.Fatalf("got %v, want nil", *got)
			case tt.want != nil && got == nil:
				t.Fatalf("got nil, want %v", *tt.want)
			case tt.want != nil && !got.Equal(*tt.want):
				t.Errorf("got %v, want %v", *got, *tt.want)
			}
		})
	}
}

func TestItemEdgeAccessors(t *testing.T) {
	it := Item{
		SellingPlaces: []SellingPlace{{ID: "p1", Name: "Whatnot"}, {ID: "p2", Name: "eBay"}},
		Labels:        []Label{{ID: "l1", Name: "Glassware"}},
	}

	if got := it.SellingPlaceIDs(); len(got) != 2 || got[0] != "p1" || got[1] != "p2" {
		t.Errorf("SellingPlaceIDs() = %v", got)
	}
	if got := it.SellingPlaceNames(); got[0] != "Whatnot" || got[1] != "eBay" {
		t.Errorf("SellingPlaceNames() = %v", got)
	}
	if got := it.LabelIDs(); len(got) != 1 || got[0] != "l1" {
		t.Errorf("LabelIDs() = %v", got)
	}
	if got := it.LabelNames(); len(got) != 1 || got[0] != "Glassware" {
		t.Errorf("LabelNames() = %v", got)
	}

	// The wire contract is an empty array, never null, so the accessors must
	// not return a nil slice for an item with no edges.
	empty := Item{}
	if empty.LabelIDs() == nil || empty.SellingPlaceNames() == nil {
		t.Error("accessors must return empty, non-nil slices")
	}
}
