// Package linear provides Linear instance database persistence.
package linear

import (
	"database/sql"

	"github.com/uptrace/opentelemetry-go-extra/otelzap"
)

// Service represents the Linear database service.
type Service struct {
	DB         *sql.DB
	Logger     *otelzap.Logger
	AESHashKey string
}
