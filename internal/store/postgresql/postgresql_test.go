package postgresql

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"
)

func TestStorePG_CreateRoomById(t *testing.T) {
	ctx := context.Background()

	id, err := uuid.NewUUID()
	assert.NoError(t, err)

	type testRow struct {
		name    string
		id      string
		setup   func(m *mocker, s *StorePG, t *testRow)
		wantErr assert.ErrorAssertionFunc
	}

	tests := []testRow{
		{
			name: "success",
			id:   id.String(),
			setup: func(m *mocker, s *StorePG, t *testRow) {
				args := pgx.NamedArgs{
					"id": t.id,
				}

				m.conn.ExpectExec(`insert into rooms \(id\) values \(@id\)`).
					WithArgs(args).
					WillReturnResult(pgxmock.NewResult("INSERT", 1)).
					WillReturnError(nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "pg_error_1",
			id:   id.String(),
			setup: func(m *mocker, s *StorePG, t *testRow) {
				args := pgx.NamedArgs{
					"id": t.id,
				}

				m.conn.ExpectExec(`insert into rooms \(id\) values \(@id\)`).
					WithArgs(args).
					WillReturnError(assert.AnError)
			},
			wantErr: assert.Error,
		},
		{
			name: "no_insert",
			id:   id.String(),
			setup: func(m *mocker, s *StorePG, t *testRow) {
				args := pgx.NamedArgs{
					"id": t.id,
				}

				m.conn.ExpectExec(`insert into rooms \(id\) values \(@id\)`).
					WithArgs(args).
					WillReturnResult(pgxmock.NewResult("INSERT", 0))
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := newMocker()
			assert.NoError(t, err)

			r := m.storePG()
			if tt.setup != nil {
				tt.setup(m, r, &tt)
			}

			tt.wantErr(t, r.CreateRoomById(ctx, tt.id), "CreateRoomById() error")
		})
	}
}

func TestStorePG_DeleteRoomById(t *testing.T) {
	ctx := context.Background()

	id, err := uuid.NewUUID()
	assert.NoError(t, err)

	type testRow struct {
		name    string
		id      string
		setup   func(m *mocker, s *StorePG, t *testRow)
		wantErr assert.ErrorAssertionFunc
	}
	tests := []testRow{
		{
			name: "success",
			id:   id.String(),
			setup: func(m *mocker, s *StorePG, t *testRow) {
				args := pgx.NamedArgs{
					"id": t.id,
				}

				m.conn.ExpectExec(`delete from rooms where id = \(@id\)`).
					WithArgs(args).
					WillReturnResult(pgxmock.NewResult("DELETE", 1)).
					WillReturnError(nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "pg_error_1",
			id:   id.String(),
			setup: func(m *mocker, s *StorePG, t *testRow) {
				args := pgx.NamedArgs{
					"id": t.id,
				}

				m.conn.ExpectExec(`delete from rooms where id = \(@id\)`).
					WithArgs(args).
					WillReturnError(assert.AnError)
			},
			wantErr: assert.Error,
		},
		{
			name: "no_insert",
			id:   id.String(),
			setup: func(m *mocker, s *StorePG, t *testRow) {
				args := pgx.NamedArgs{
					"id": t.id,
				}

				m.conn.ExpectExec(`delete from rooms where id = \(@id\)`).
					WithArgs(args).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := newMocker()
			assert.NoError(t, err)

			r := m.storePG()
			if tt.setup != nil {
				tt.setup(m, r, &tt)
			}

			tt.wantErr(t, r.DeleteRoomById(ctx, tt.id), "DeleteRoomById() error")
		})
	}
}
