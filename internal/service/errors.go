package service

import (
	"database/sql"
	"errors"

	"vaccineshit/internal/domain"
)

func domainConflict(message string) error {
	return domain.NewError(domain.CodeConflict, "", message)
}

func domainInvalid(message string) error {
	return domain.NewError(domain.CodeInvalidInput, "", message)
}

func notFound(entity string) error {
	return domain.NotFound(entity)
}

func requireFound(err error, entity string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return notFound(entity)
	}
	return err
}
