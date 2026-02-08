package db

type BlocksQ interface {
	New() BlocksQ
	Insert(LatestBlock) (err error)
	GetLatestBlock() (int64, error)

	UpdateLatestBlockId(block LatestBlock) error

	Transaction(f func() error) error
}

type LatestBlock struct {
	Id      int64 `structs:"-" db:"-"`
	BlockId int64 `structs:"latest_block_id" db:"latest_block_id"`
}
