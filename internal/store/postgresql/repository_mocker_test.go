package postgresql

import (
	"github.com/pashagolub/pgxmock/v2"
	"github.com/pkg/errors"
)

func newMocker() (*mocker, error) {
	var conn, err = pgxmock.NewPool()
	if err != nil {
		return nil, errors.Wrap(err, "new postgresql.connection")
	}

	return &mocker{conn: conn}, nil
}

type mocker struct {
	conn pgxmock.PgxPoolIface
}

func (m *mocker) storePG() *StorePG {
	return &StorePG{
		m.conn,
	}
}
