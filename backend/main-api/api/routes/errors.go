package routes

import (
	"errors"

	"main-api/internal/app/domain"

	"github.com/danielgtaylor/huma/v2"
)

// toHTTP turns an application error into the HTTP error the client sees. The
// message comes from the domain, which owns the wording; this only picks the
// status code. Anything that is not a domain error, or is internal, is
// returned as-is so huma answers 500 without leaking the detail.
func toHTTP(err error) error {
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) {
		return err
	}

	switch domainErr.Kind {
	case domain.KindInvalid:
		return huma.Error400BadRequest(domainErr.Msg)
	case domain.KindNotFound:
		return huma.Error404NotFound(domainErr.Msg)
	case domain.KindConflict:
		return huma.Error409Conflict(domainErr.Msg)
	case domain.KindTooLarge:
		return huma.Error413RequestEntityTooLarge(domainErr.Msg)
	case domain.KindUnavailable:
		return huma.Error503ServiceUnavailable(domainErr.Msg)
	default:
		return err
	}
}
