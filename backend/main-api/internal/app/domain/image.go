package domain

import (
	"path/filepath"
	"strings"
	"time"
)

// MaxImageBytes caps a single image upload at 15 MB — generous for phone
// photos, small enough to bound request bodies.
const MaxImageBytes = 15 << 20

// PresignTTL is how long an image view URL stays valid.
const PresignTTL = 15 * time.Minute

// allowedImageTypes maps an accepted content type to its file extension.
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// ItemImage is one stored photo of an item.
type ItemImage struct {
	ID           string
	ItemID       string
	UploadBucket string
	UploadKey    string
	Filename     *string
	ContentType  *string
	DisplayOrder int
}

// NewItemImage is everything needed to record an uploaded photo.
type NewItemImage struct {
	ItemID       string
	UploadBucket string
	UploadKey    string
	Filename     *string
	ContentType  string
	DisplayOrder int
}

// ValidateUpload checks a proposed upload against the size cap and the
// accepted types, returning the extension to store it under.
func ValidateUpload(size int64, contentType string) (string, error) {
	if size > MaxImageBytes {
		return "", TooLarge("image exceeds 15 MB")
	}
	ext, ok := allowedImageTypes[strings.ToLower(contentType)]
	if !ok {
		return "", Invalidf("unsupported image type %q", contentType)
	}
	return ext, nil
}

// ObjectKey is where an item's photo lives in the bucket. unique is an
// opaque per-object id supplied by the caller, keeping id generation out of
// the domain.
func ObjectKey(itemID, unique, defaultExt, filename string) string {
	return "items/" + itemID + "/" + unique + extOrDefault(defaultExt, filename)
}

// extOrDefault prefers the uploaded filename's own extension, falling back to
// the one implied by the content type.
func extOrDefault(defaultExt, filename string) string {
	if e := strings.ToLower(filepath.Ext(filename)); e != "" {
		return e
	}
	return defaultExt
}

// FilenameOrNil drops an empty filename so it is never stored as "".
func FilenameOrNil(name string) *string {
	if name == "" {
		return nil
	}
	return &name
}

// Cover is the photo a list view shows for an item: the first in display
// order. Images are expected in display order, as the repository returns them.
func Cover(images []ItemImage) (ItemImage, bool) {
	if len(images) == 0 {
		return ItemImage{}, false
	}
	return images[0], true
}
