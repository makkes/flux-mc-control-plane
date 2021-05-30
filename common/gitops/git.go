package gitops

import "context"

type Repository interface {
	Clone(context.Context, string) error
	Remove(path string) error
	Add(path string) error
	Commit(msg, name, email string) error
	Push(context.Context) error
}
