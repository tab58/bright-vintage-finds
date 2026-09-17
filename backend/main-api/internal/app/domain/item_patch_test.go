package domain

import "testing"

func TestClearable(t *testing.T) {
	// The PWA sends an absent field to mean "leave it", an empty string to
	// mean "clear it", and anything else to mean "store this". Getting the
	// empty-string case wrong stores "" in a unique column.
	tests := []struct {
		name  string
		in    *string
		want  FieldAction
		value string
	}{
		{name: "absent leaves the column alone", in: nil, want: FieldLeave},
		{name: "empty string clears", in: ptr(""), want: FieldClear},
		{name: "value sets", in: ptr("WN-42"), want: FieldSet, value: "WN-42"},
		{name: "whitespace is a value, not a clear", in: ptr(" "), want: FieldSet, value: " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Clearable(tt.in)
			if got.Action != tt.want {
				t.Fatalf("Action = %d, want %d", got.Action, tt.want)
			}
			if got.Value != tt.value {
				t.Errorf("Value = %q, want %q", got.Value, tt.value)
			}
		})
	}
}

func TestClearableStored(t *testing.T) {
	tests := []struct {
		name    string
		in      *string
		wantOK  bool
		wantVal string
	}{
		{name: "absent stores nothing", in: nil},
		{name: "empty stores nothing on create", in: ptr("")},
		{name: "value stores", in: ptr("note"), wantOK: true, wantVal: "note"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, ok := Clearable(tt.in).Stored()
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				if v != nil {
					t.Errorf("value = %q, want nil", *v)
				}
				return
			}
			if *v != tt.wantVal {
				t.Errorf("value = %q, want %q", *v, tt.wantVal)
			}
		})
	}
}

func TestClampPageSize(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"unset takes the default", 0, CatalogPageDefault},
		{"negative takes the default", -5, CatalogPageDefault},
		{"in range passes through", 10, 10},
		{"at the cap passes through", CatalogPageMax, CatalogPageMax},
		{"over the cap is clamped", CatalogPageMax + 1, CatalogPageMax},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClampPageSize(tt.in); got != tt.want {
				t.Errorf("ClampPageSize(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }
