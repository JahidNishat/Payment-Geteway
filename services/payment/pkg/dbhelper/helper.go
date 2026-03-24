package dbhelper

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

func IsDuplicate(err error) bool {
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return true
		}
	}
	return false
}

func IsNotFound(err error) bool {
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	return false
}
