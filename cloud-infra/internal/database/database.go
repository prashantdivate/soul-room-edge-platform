package database

import "context"

type Migrator interface {
	Apply(context.Context, string) error
	Version(context.Context) (string, error)
}
