package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"main-api/internal/app/domain"
)

func TestUploadStoresTheObjectThenTheRow(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	item, _ := h.app.CreateItem(ctx, ItemInput{Name: "vase"})

	got, err := h.app.UploadItemImage(ctx, UploadInput{
		ItemID:      item.Item.ID,
		Filename:    "IMG_0001.HEIC.jpg",
		ContentType: "image/jpeg",
		Size:        1024,
		Body:        strings.NewReader("photo bytes"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	wantKey := "items/" + item.Item.ID + "/obj1.jpg"
	if got.UploadKey != wantKey {
		t.Errorf("key = %q, want %q", got.UploadKey, wantKey)
	}
	if got.UploadBucket != "uploads" {
		t.Errorf("bucket = %q, want the configured upload bucket", got.UploadBucket)
	}
	if body := string(h.store.Objects["uploads/"+wantKey]); body != "photo bytes" {
		t.Errorf("stored %q, want the whole body", body)
	}
	if got.DisplayOrder != 0 {
		t.Errorf("display order = %d, want 0 for the first photo", got.DisplayOrder)
	}
	if got.Filename == nil || *got.Filename != "IMG_0001.HEIC.jpg" {
		t.Errorf("filename = %v", got.Filename)
	}

	// The next upload goes after the current last photo.
	second, err := h.app.UploadItemImage(ctx, UploadInput{
		ItemID: item.Item.ID, Filename: "b.png", ContentType: "image/png",
		Size: 10, Body: strings.NewReader("more"),
	})
	if err != nil {
		t.Fatalf("second Upload: %v", err)
	}
	if second.DisplayOrder != 1 {
		t.Errorf("display order = %d, want 1", second.DisplayOrder)
	}
	if !strings.HasSuffix(second.UploadKey, ".png") {
		t.Errorf("key = %q, want the filename's extension", second.UploadKey)
	}
}

func TestUploadRejectsBadPayloads(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		size        int64
		contentType string
		wantErr     string
		wantKind    domain.Kind
	}{
		{
			name: "over the size cap", size: domain.MaxImageBytes + 1, contentType: "image/jpeg",
			wantErr: "image exceeds 15 MB", wantKind: domain.KindTooLarge,
		},
		{
			name: "a type the shop cannot display", size: 10, contentType: "application/pdf",
			wantErr: `unsupported image type "application/pdf"`, wantKind: domain.KindInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			item, _ := h.app.CreateItem(ctx, ItemInput{Name: "vase"})

			_, err := h.app.UploadItemImage(ctx, UploadInput{
				ItemID: item.Item.ID, Filename: "f", ContentType: tt.contentType,
				Size: tt.size, Body: strings.NewReader("x"),
			})
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("err = %v, want %q", err, tt.wantErr)
			}
			if domain.KindOf(err) != tt.wantKind {
				t.Errorf("kind = %d, want %d", domain.KindOf(err), tt.wantKind)
			}
			// Nothing reached the bucket.
			if len(h.store.Objects) != 0 {
				t.Errorf("stored %d objects for a rejected upload", len(h.store.Objects))
			}
		})
	}
}

func TestUploadNeedsTheItemToExist(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)

	_, err := h.app.UploadItemImage(ctx, UploadInput{
		ItemID: "nope", Filename: "a.jpg", ContentType: "image/jpeg",
		Size: 10, Body: strings.NewReader("x"),
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want not found", err)
	}
	// Storage is not spent on an item that is not there.
	if len(h.store.Objects) != 0 {
		t.Errorf("stored %d objects for an unknown item", len(h.store.Objects))
	}
}

func TestUploadWithoutStorageIsUnavailable(t *testing.T) {
	ctx := context.Background()
	h := newHarnessWithoutStorage(t)
	item, _ := h.app.CreateItem(ctx, ItemInput{Name: "vase"})

	_, err := h.app.UploadItemImage(ctx, UploadInput{
		ItemID: item.Item.ID, Filename: "a.jpg", ContentType: "image/jpeg",
		Size: 10, Body: strings.NewReader("x"),
	})
	if err == nil || err.Error() != "object storage is not configured" {
		t.Fatalf("err = %v, want the storage message", err)
	}
	if domain.KindOf(err) != domain.KindUnavailable {
		t.Errorf("kind = %d, want unavailable", domain.KindOf(err))
	}
}

func TestListImagesReturnsFreshURLsInDisplayOrder(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	item, _ := h.app.CreateItem(ctx, ItemInput{Name: "vase"})
	h.addPhoto(item.Item.ID, "items/v/second.jpg", 1)
	h.addPhoto(item.Item.ID, "items/v/first.jpg", 0)

	got, err := h.app.ListItemImages(ctx, item.Item.ID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("photos = %d, want 2", len(got))
	}
	if got[0].Image.UploadKey != "items/v/first.jpg" {
		t.Errorf("order = %v", got[0].Image.UploadKey)
	}
	if got[0].URL != "https://storage.test/uploads/items/v/first.jpg" {
		t.Errorf("url = %q", got[0].URL)
	}
}

func TestPresignFailureIsReportedNotSwallowed(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	item, _ := h.app.CreateItem(ctx, ItemInput{Name: "vase"})
	h.addPhoto(item.Item.ID, "items/v/a.jpg", 0)
	cause := errors.New("credentials expired")
	h.store.PresignErr = cause

	// A cover URL that cannot be minted is a failure, not a missing cover:
	// silently dropping it would show the owner an item with no photo.
	if _, err := h.app.GetItem(ctx, item.Item.ID); !errors.Is(err, cause) {
		t.Errorf("Get = %v, want the presign failure", err)
	}
	if _, err := h.app.ListItemImages(ctx, item.Item.ID); !errors.Is(err, cause) {
		t.Errorf("List = %v, want the presign failure", err)
	}
}

func TestListItemImagesWithoutStorageIsUnavailable(t *testing.T) {
	ctx := context.Background()
	h := newHarnessWithoutStorage(t)

	// Without storage the photo routes are not even registered, but the use
	// case must still refuse rather than presign against a nil store.
	_, err := h.app.ListItemImages(ctx, "item-1")
	if err == nil {
		t.Fatal("ListItemImages = nil error without storage")
	}
	if domain.KindOf(err) != domain.KindUnavailable {
		t.Errorf("kind = %v, want unavailable", domain.KindOf(err))
	}
}
