package repository

import (
	"errors"

	"car-rental-system/pkg/apperror"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func mapPostgresError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperror.Wrap(apperror.ErrConflict, err)
		case "23503":
			return apperror.Wrap(apperror.ErrBadRequest, err)
		case "23P01":
			return apperror.ErrDoubleBooking
		}
	}

	return err
}

func isPostgresConstraint(err error, code string, constraintNames ...string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != code {
		return false
	}
	if len(constraintNames) == 0 {
		return true
	}

	for _, constraintName := range constraintNames {
		if pgErr.ConstraintName == constraintName {
			return true
		}
	}

	return false
}
