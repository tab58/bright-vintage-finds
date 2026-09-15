package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"main-api/db/generated"
	"main-api/db/generated/item"
	"main-api/db/generated/itemimage"

	"github.com/danielgtaylor/huma/v2"
	"github.com/segmentio/ksuid"
)

// maxImageBytes caps a single image upload at 15 MB — generous for phone
// photos, small enough to bound request bodies.
const maxImageBytes = 15 << 20

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type uploadImageInput struct {
	ID string `path:"id" doc:"Item ID"`
	RawBody huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" contentType:"image/jpeg,image/png,image/webp" required:"true"`
	}]
}

type uploadImageOutput struct {
	Body *ItemImageOutput `json:"body"`
}

// ItemImageOutput is an uploaded picture reference.
type ItemImageOutput struct {
	ID           string `json:"id"`
	UploadKey    string `json:"upload_key"`
	UploadBucket string `json:"upload_bucket"`
	Filename     string `json:"filename,omitempty"`
	ContentType  string `json:"content_type,omitempty"`
	DisplayOrder int    `json:"display_order"`
	// URL is the presigned view URL, filled per-request.
	URL string `json:"url,omitempty" doc:"Presigned GET URL, valid briefly"`
}

// presignTTL is how long an image view URL stays valid.
const presignTTL = 15 * time.Minute

// imageOutputWithURL maps an ItemImage row to its API shape with a presigned
// view URL.
func imageOutputWithURL(ctx context.Context, deps *AppDeps, ii *generated.ItemImage) (*ItemImageOutput, error) {
	out := &ItemImageOutput{
		ID:           ii.ID,
		UploadKey:    ii.UploadKey,
		UploadBucket: ii.UploadBucket,
		Filename:     derefString(ii.Filename),
		ContentType:  derefString(ii.ContentType),
		DisplayOrder: ii.DisplayOrder,
	}
	url, err := deps.Store.PresignGetObject(ctx, ii.UploadBucket, ii.UploadKey, presignTTL)
	if err != nil {
		return nil, fmt.Errorf("presigning image %s: %w", ii.ID, err)
	}
	if deps.S3PublicEndpoint != "" && deps.S3InternalEndpoint != "" {
		url = strings.Replace(url, deps.S3InternalEndpoint, deps.S3PublicEndpoint, 1)
	}
	out.URL = url
	return out, nil
}

// listImagesOutput returns the item's images with fresh presigned GET URLs.
type listImagesOutput struct {
	Body []*ItemImageOutput `json:"body"`
}

// registerItemImageRoutes registers image list and upload routes. They are
// only registered when object storage is configured.
func registerItemImageRoutes(api huma.API, deps *AppDeps) {
	db := deps.DB
	client := db.GetDBFromContext(nil)

	huma.Register(api, huma.Operation{
		OperationID: "list-item-images",
		Method:      http.MethodGet,
		Path:        "/admin/items/{id}/images",
		Summary:     "List an item's images with view URLs",
	}, func(ctx context.Context, in *itemIDInput) (*listImagesOutput, error) {
		imgs, err := client.ItemImage.Query().
			Where(itemimage.HasItemWith(item.ID(in.ID)), itemimage.DeletedAtIsNil()).
			Order(generated.Asc(itemimage.FieldDisplayOrder)).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing item images: %w", err)
		}
		out := &listImagesOutput{Body: make([]*ItemImageOutput, 0, len(imgs))}
		for _, ii := range imgs {
			o, err := imageOutputWithURL(ctx, deps, ii)
			if err != nil {
				return nil, err
			}
			out.Body = append(out.Body, o)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "upload-item-image",
		Method:      http.MethodPost,
		Path:        "/admin/items/{id}/images",
		Summary:     "Upload a picture for an item",
	}, func(ctx context.Context, in *uploadImageInput) (*uploadImageOutput, error) {
		if deps.Store == nil {
			return nil, huma.Error503ServiceUnavailable("object storage is not configured")
		}

		file := in.RawBody.Data().File
		if file.Size > maxImageBytes {
			return nil, huma.Error413RequestEntityTooLarge("image exceeds 15 MB")
		}
		ext, ok := allowedImageTypes[strings.ToLower(file.ContentType)]
		if !ok {
			return nil, huma.Error400BadRequest(fmt.Sprintf("unsupported image type %q", file.ContentType))
		}

		// The item must exist before we spend storage on it.
		it, err := client.Item.Query().
			Where(item.ID(in.ID), item.DeletedAtIsNil()).
			Only(ctx)
		if generated.IsNotFound(err) {
			return nil, huma.Error404NotFound("item not found")
		}
		if err != nil {
			return nil, fmt.Errorf("loading item: %w", err)
		}

		// Stream the multipart file straight into storage.
		uploadKey := fmt.Sprintf("items/%s/%s%s", it.ID, ksuid.New().String(), extOrFilenameExt(ext, file.Filename))
		if err := deps.Store.UploadFile(ctx, deps.S3UploadBucket, uploadKey, &file); err != nil {
			return nil, fmt.Errorf("uploading image to storage: %w", err)
		}

		// Next display_order: after the current last image.
		count, err := client.ItemImage.Query().
			Where(itemimage.HasItemWith(item.ID(it.ID)), itemimage.DeletedAtIsNil()).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("counting images: %w", err)
		}

		img, err := client.ItemImage.Create().
			SetItemID(it.ID).
			SetUploadBucket(deps.S3UploadBucket).
			SetUploadKey(uploadKey).
			SetNillableFilename(fileNamePtr(file.Filename)).
			SetContentType(file.ContentType).
			SetDisplayOrder(count).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("recording image: %w", err)
		}

		return &uploadImageOutput{Body: &ItemImageOutput{
			ID:           img.ID,
			UploadKey:    img.UploadKey,
			UploadBucket: img.UploadBucket,
			Filename:     derefString(img.Filename),
			ContentType:  derefString(img.ContentType),
			DisplayOrder: img.DisplayOrder,
		}}, nil
	})
}

func fileNamePtr(name string) *string {
	if name == "" {
		return nil
	}
	return &name
}
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
func extOrFilenameExt(defaultExt, filename string) string {
	if e := strings.ToLower(filepath.Ext(filename)); e != "" {
		return e
	}
	return defaultExt
}