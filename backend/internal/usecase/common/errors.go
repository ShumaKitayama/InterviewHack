package common

import (
	"errors"

	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/port/repository"
)

// MapRepoErr converts repository sentinel errors to domain errors.
// Rows of other users are reported as NOT_FOUND (7.4).
func MapRepoErr(err error, resource string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNotFound):
		return domainerror.ErrNotFound(resource)
	case errors.Is(err, repository.ErrInvalidCursor):
		return domainerror.ErrValidation("cursor が不正です")
	case errors.Is(err, repository.ErrConflict):
		return domainerror.ErrConflict(resource + " が重複しています")
	}
	return err
}

// EnsureOwned returns NOT_FOUND unless every id is owned by the user.
func EnsureOwned(count int, ids []string, resource string) error {
	if count != len(ids) {
		return domainerror.ErrNotFound(resource)
	}
	return nil
}
