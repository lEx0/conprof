package storage

import (
	"errors"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/ncw/swift"
	"io"
	"net/url"
	"time"
)

type (
	Options struct {
		URL     string
		Timeout time.Duration
		Bucket  string
	}
	SwiftStorage struct {
		bucket string
		conn   *swift.Connection
		logger log.Logger
	}
)

func NewSwiftStorage(options Options, logger log.Logger) (c *SwiftStorage, err error) {
	var uri *url.URL

	if options.Bucket == "" {
		return nil, errors.New("bucket is empty")
	}

	conn := &swift.Connection{
		Timeout: options.Timeout,
	}

	if uri, err = url.Parse(options.URL); err != nil {
		return
	}

	conn.UserName = uri.User.Username()
	if key, exists := uri.User.Password(); exists {
		conn.ApiKey = key
	}

	// сбрасываем данные авторизации, что бы получить чистый URL
	uri.User = nil
	conn.AuthUrl = uri.String()

	if err = conn.Authenticate(); err != nil {
		return
	}

	return &SwiftStorage{
		conn:   conn,
		bucket: options.Bucket,
		logger: logger,
	}, err
}

func (s *SwiftStorage) Upload(name string, r io.Reader) (err error) {
	_, err = s.conn.ObjectPut(s.bucket, name, r, true, "", "application/octet-stream", swift.Headers{})
	return
}

func (s *SwiftStorage) Delete(name string) error {
	return s.conn.ObjectDelete(s.bucket, name)
}

func (s *SwiftStorage) Get(name string, output io.Writer) (err error) {
	_, err = s.conn.ObjectGet(s.bucket, name, output, true, swift.Headers{})
	return
}

func (s *SwiftStorage) PruneOldest(date time.Time) error {
	removeCh := make(chan swift.Object, 10)
	defer close(removeCh)

	go func() {
		for object := range removeCh {
			if object.LastModified.Before(date) {
				level.Info(s.logger).Log("remove file", object.Name)
				if err := s.Delete(object.Name); err != nil {
					//noinspection GoUnhandledErrorResult
					level.Error(s.logger).Log("cannot remove object", object.Name, err)
				}
			}
		}
	}()

	return s.conn.ObjectsWalk(
		s.bucket, &swift.ObjectsOpts{Limit: 50}, func(opts *swift.ObjectsOpts) (interface{}, error) {
			objects, err := s.conn.Objects(s.bucket, opts)

			if err == nil {
				for _, object := range objects {
					removeCh <- object
				}
			}

			return objects, err
		},
	)
}
