package storage

import (
	"errors"
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
	}
)

func NewSwiftStorage(options Options) (c *SwiftStorage, err error) {
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
	return s.conn.ObjectsWalk(
		s.bucket, &swift.ObjectsOpts{Limit: 50}, func(opts *swift.ObjectsOpts) (interface{}, error) {
			objects, err := s.conn.Objects(s.bucket, opts)

			if err == nil {
				for _, object := range objects {
					if object.LastModified.Before(date) {
						if err = s.Delete(object.Name); err != nil {
							return nil, err
						}
					}
				}
			}

			return objects, err
		},
	)
}
