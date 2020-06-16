package rtsb

import (
	"bytes"
	"fmt"
	"github.com/go-kit/kit/log"
	"math/rand"
	"strconv"

	"github.com/conprof/tsdb"
	"github.com/conprof/tsdb/labels"
	"github.com/lEx0/conprof/rtsb/storage"
)

type dbAppender struct {
	appender tsdb.Appender
	storage  storage.Storage
	logger   log.Logger
}

func (c *dbAppender) Add(l labels.Labels, t int64, v []byte) (i uint64, err error) {
	name := fmt.Sprintf("%d-%s.pprof", t, strconv.FormatUint(rand.Uint64(), 16))

	if err = c.storage.Upload(name, bytes.NewBuffer(v)); err != nil {
		return
	}

	if i, err = c.appender.Add(l, t, []byte(name)); err != nil {
		if err := c.storage.Delete(name); err != nil {
			//noinspection GoUnhandledErrorResult
			c.logger.Log("cant delete file: %s\n", name)
		}
	}

	return
}

func (c *dbAppender) AddFast(ref uint64, t int64, v []byte) (err error) {
	name := fmt.Sprintf("%d-%s.pprof", t, strconv.FormatUint(rand.Uint64(), 16))

	if err = c.storage.Upload(name, bytes.NewBuffer(v)); err != nil {
		return
	}

	if err = c.appender.AddFast(ref, t, []byte(name)); err != nil {
		if err := c.storage.Delete(name); err != nil {
			//noinspection GoUnhandledErrorResult
			c.logger.Log("cannt delete file: %s\n", name)
		}
	}

	return
}

func (c *dbAppender) Commit() error {
	return c.appender.Commit()
}

func (c *dbAppender) Rollback() error {
	return c.appender.Rollback()
}
