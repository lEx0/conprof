package storage

import (
	"io"
	"time"
)

type Storage interface {
	Upload(name string, r io.Reader) error
	Delete(name string) error
	PruneOldest(date time.Time) error
	Get(name string, output io.Writer) error
}
