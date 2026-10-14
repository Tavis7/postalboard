package config

import (
	"github.com/tavis7/postalboard/internal/database"
)

type Config struct {
	PasswordMinLength int32
	PasswordMaxLength int32
	DbQueries         *database.Queries
}
