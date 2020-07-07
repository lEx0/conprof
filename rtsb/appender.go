package rtsb

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/cespare/xxhash"
	"github.com/conprof/tsdb"
	"github.com/conprof/tsdb/labels"
	"github.com/go-kit/kit/log"
	"github.com/lEx0/conprof/rtsb/storage"
)

type dbAppender struct {
	appender tsdb.Appender
	storage  storage.Storage
	logger   log.Logger
}

func (c *dbAppender) Add(l labels.Labels, t int64, v []byte) (i uint64, err error) {
	// т.к. rand.Uint64() возвращает число от 0 до 18446744073709551615, можем вполне получить 0
	// а нам потом надо по бакетам распиливать, так что дополнительно хешируем через xxhash
	// что бы получить строку с фиксированной длинной
	name := fmt.Sprintf("%d-%d.pprof", xxhash.Sum64([]byte(strconv.FormatUint(rand.Uint64(), 16))), t)

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
	// т.к. rand.Uint64() возвращает число от 0 до 18446744073709551615, можем вполне получить 0
	// а нам потом надо по бакетам секционировать, так что дополнительно хешируем через xxhash
	// что бы получить строку с фиксированной длинной
	name := fmt.Sprintf("%d-%d.pprof", xxhash.Sum64([]byte(strconv.FormatUint(rand.Uint64(), 16))), t)

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
