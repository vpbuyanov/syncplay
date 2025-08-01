package postgresql

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

type repository interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type StorePG struct {
	db repository
}

func NewRepos(db *pgxpool.Pool) *StorePG {
	return &StorePG{
		db,
	}
}

func (s *StorePG) CreateRoomById(ctx context.Context, id string) error {
	args := pgx.NamedArgs{
		"id": id,
	}

	exec, err := s.db.Exec(ctx, `insert into rooms (id) values (@id)`, args)
	if err != nil {
		return errors.Wrap(err, "insert room in pg")
	}

	if exec.RowsAffected() != 1 {
		return errors.New("no insert room")
	}

	return nil
}

func (s *StorePG) DeleteRoomById(ctx context.Context, id string) error {
	args := pgx.NamedArgs{
		"id": id,
	}

	exec, err := s.db.Exec(ctx, "delete from rooms where id = (@id)", args)
	if err != nil {
		return errors.Wrap(err, "delete room in pg")
	}

	if exec.RowsAffected() == 0 {
		return errors.New("no delete room")
	}

	return nil
}
