package pg

import (
	"database/sql"

	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
	"github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/kit/pgdb"
)

const (
	blocksTable        = "latest_block"
	latestBlockIdField = "latest_block_id"
	idField            = "id"
	idIndex            = 1
)

type blocksQ struct {
	db       *pgdb.DB
	selector squirrel.SelectBuilder
}

func (d *blocksQ) Insert(block db.LatestBlock) error {
	stmt := squirrel.Insert(blocksTable).
		Columns(idField, latestBlockIdField).
		Values(idIndex, block.BlockId).
		Suffix("ON CONFLICT (" + idField + ") DO UPDATE SET " +
			latestBlockIdField + " = EXCLUDED." + latestBlockIdField)

	return errors.Wrap(d.db.Exec(stmt), "failed to upsert latest block")
}

func (d *blocksQ) GetLatestBlock() (int64, error) {
	stmt := squirrel.Select(latestBlockIdField).
		From(blocksTable).
		Where(squirrel.Eq{idField: idIndex})

	var id int64
	err := d.db.Get(&id, stmt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}

	return id, errors.Wrap(err, "failed to fetch latest block")
}

func (d *blocksQ) UpdateLatestBlockId(block db.LatestBlock) error {
	stmt := squirrel.Update(blocksTable).
		SetMap(map[string]interface{}{
			latestBlockIdField: block.BlockId,
		}).Where(squirrel.Eq{idField: idIndex})

	return errors.Wrap(d.db.Exec(stmt), "failed to update latest block id")
}

func (d *blocksQ) Transaction(f func() error) error {
	return d.db.Transaction(f)
}

func (d *blocksQ) New() db.BlocksQ {
	return NewBlocksQ(d.db.Clone())
}

func NewBlocksQ(db *pgdb.DB) db.BlocksQ {
	return &blocksQ{
		db:       db.Clone(),
		selector: squirrel.Select("*").From(blocksTable),
	}
}
