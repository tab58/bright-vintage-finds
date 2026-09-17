package domain

import "testing"

func TestValidateUpload(t *testing.T) {
	tests := []struct {
		name        string
		size        int64
		contentType string
		wantExt     string
		wantErr     string
		wantKind    Kind
	}{
		{name: "jpeg", size: 1000, contentType: "image/jpeg", wantExt: ".jpg"},
		{name: "png", size: 1000, contentType: "image/png", wantExt: ".png"},
		{name: "webp", size: 1000, contentType: "image/webp", wantExt: ".webp"},
		{name: "content type is matched case-insensitively", size: 1, contentType: "IMAGE/JPEG", wantExt: ".jpg"},
		{name: "at the cap is allowed", size: MaxImageBytes, contentType: "image/png", wantExt: ".png"},
		{
			name: "over the cap", size: MaxImageBytes + 1, contentType: "image/png",
			wantErr: "image exceeds 15 MB", wantKind: KindTooLarge,
		},
		{
			name: "unsupported type", size: 10, contentType: "image/gif",
			wantErr: `unsupported image type "image/gif"`, wantKind: KindInvalid,
		},
		{
			// Size is checked first: an oversized GIF reports the size, matching
			// the order the handler used.
			name: "size is checked before type", size: MaxImageBytes + 1, contentType: "image/gif",
			wantErr: "image exceeds 15 MB", wantKind: KindTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, err := ValidateUpload(tt.size, tt.contentType)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("want error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Errorf("message = %q, want %q", err.Error(), tt.wantErr)
				}
				if KindOf(err) != tt.wantKind {
					t.Errorf("kind = %d, want %d", KindOf(err), tt.wantKind)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ext != tt.wantExt {
				t.Errorf("ext = %q, want %q", ext, tt.wantExt)
			}
		})
	}
}

func TestObjectKey(t *testing.T) {
	tests := []struct {
		name       string
		defaultExt string
		filename   string
		want       string
	}{
		{
			name:       "the uploaded filename's extension wins",
			defaultExt: ".jpg", filename: "shot.PNG",
			want: "items/itm1/uniq.png",
		},
		{
			name:       "no filename falls back to the content type",
			defaultExt: ".webp", filename: "",
			want: "items/itm1/uniq.webp",
		},
		{
			name:       "extensionless filename falls back too",
			defaultExt: ".jpg", filename: "IMG_0001",
			want: "items/itm1/uniq.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ObjectKey("itm1", "uniq", tt.defaultExt, tt.filename); got != tt.want {
				t.Errorf("ObjectKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFilenameOrNil(t *testing.T) {
	if got := FilenameOrNil(""); got != nil {
		t.Errorf("empty filename should not be stored, got %q", *got)
	}
	if got := FilenameOrNil("a.jpg"); got == nil || *got != "a.jpg" {
		t.Errorf("FilenameOrNil(a.jpg) = %v", got)
	}
}

func TestCover(t *testing.T) {
	// The repository returns photos in display order, so the cover is the
	// first element; nothing here re-sorts.
	images := []ItemImage{{ID: "i1", DisplayOrder: 0}, {ID: "i2", DisplayOrder: 1}}

	got, ok := Cover(images)
	if !ok || got.ID != "i1" {
		t.Errorf("Cover() = %v, %v; want i1, true", got.ID, ok)
	}
	if _, ok := Cover(nil); ok {
		t.Error("an item with no photos has no cover")
	}
}
