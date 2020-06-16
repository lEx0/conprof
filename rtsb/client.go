package rtsb

import (
	"github.com/conprof/tsdb"
	"github.com/go-kit/kit/log"
	"github.com/lEx0/conprof/rtsb/storage"
	"time"
)

type RemoteTSDB struct {
	logger  log.Logger
	storage storage.Storage
	db      *tsdb.DB
}

func NewRemoteTSDB(storage storage.Storage, db *tsdb.DB) *RemoteTSDB {
	return &RemoteTSDB{
		storage: storage,
		db:      db,
	}
}

func (c *RemoteTSDB) Appender() tsdb.Appender {
	return &dbAppender{
		c.db.Appender(),
		c.storage,
		c.logger,
	}
}

func (c *RemoteTSDB) Clean(retention time.Duration) error {
	return c.storage.PruneOldest(time.Now().Add(-retention))
}
