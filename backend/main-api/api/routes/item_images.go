package routes

import (
	"context"
	"net/http"

	"main-api/internal/app"
	"main-api/internal/app/domain"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

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

// imageOutput maps a stored photo without a URL, as the upload response does:
// the client already has the bytes it just sent.
func imageOutput(img domain.ItemImage) *ItemImageOutput {
	return &ItemImageOutput{
		ID:           img.ID,
		UploadKey:    img.UploadKey,
		UploadBucket: img.UploadBucket,
		Filename:     derefString(img.Filename),
		ContentType:  derefString(img.ContentType),
		DisplayOrder: img.DisplayOrder,
	}
}

// imageViewOutput maps a photo together with the URL to fetch it from.
func imageViewOutput(v app.ImageView) *ItemImageOutput {
	out := imageOutput(v.Image)
	out.URL = v.URL
	return out
}

// listImagesOutput returns the item's images with fresh presigned GET URLs.
type listImagesOutput struct {
	Body []*ItemImageOutput `json:"body"`
}

// RegisterItemImageRoutes registers image list and upload routes. They are
// only registered when object storage is configured.
func RegisterItemImageRoutes(api huma.API, a *app.Application) {
	// humago buffers only 8 KiB of a multipart body in memory and spills the
	// rest to a temp file. The production image is FROM scratch and has no
	// /tmp, so every real photo failed with 422 "cannot read multipart form".
	// Buffering the whole capped upload keeps it off disk entirely.
	humago.MultipartMaxMemory = domain.MaxImageBytes

	huma.Register(api, huma.Operation{
		OperationID: "list-item-images",
		Method:      http.MethodGet,
		Path:        "/admin/items/{id}/images",
		Summary:     "List an item's images with view URLs",
	}, func(ctx context.Context, in *itemIDInput) (*listImagesOutput, error) {
		views, err := a.ListItemImages(ctx, in.ID)
		if err != nil {
			return nil, toHTTP(err)
		}
		out := &listImagesOutput{Body: make([]*ItemImageOutput, 0, len(views))}
		for _, v := range views {
			out.Body = append(out.Body, imageViewOutput(v))
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "upload-item-image",
		Method:      http.MethodPost,
		Path:        "/admin/items/{id}/images",
		Summary:     "Upload a picture for an item",
	}, func(ctx context.Context, in *uploadImageInput) (*uploadImageOutput, error) {
		file := in.RawBody.Data().File

		img, err := a.UploadItemImage(ctx, app.UploadInput{
			ItemID:      in.ID,
			Filename:    file.Filename,
			ContentType: file.ContentType,
			Size:        file.Size,
			Body:        &file,
		})
		if err != nil {
			return nil, toHTTP(err)
		}
		return &uploadImageOutput{Body: imageOutput(img)}, nil
	})
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
