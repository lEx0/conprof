package storage

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/ncw/swift"
)

type (
	Options struct {
		URL                    string
		Timeout                time.Duration
		Bucket                 string
		PartitionPrefixKeySize int
	}
	SwiftStorage struct {
		bucket  string
		conn    *swift.Connection
		logger  log.Logger
		keySize int
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
		conn:    conn,
		bucket:  options.Bucket,
		logger:  logger,
		keySize: options.PartitionPrefixKeySize,
	}, err
}

func (s *SwiftStorage) getBucketName(name string) string {
	bucket := s.bucket

	if s.keySize > 0 {
		bucket = fmt.Sprintf("%s-%s", s.bucket, name[0:s.keySize])
	}

	return bucket
}

func (s *SwiftStorage) Upload(name string, r io.Reader) (err error) {
	//noinspection GoUnhandledErrorResult
	level.Debug(s.logger).Log("upload object", name)

	_, err = s.conn.ObjectPut(s.getBucketName(name), name, r, true, "", "application/octet-stream", swift.Headers{})
	return
}

func (s *SwiftStorage) Delete(name string) error {
	//noinspection GoUnhandledErrorResult
	level.Debug(s.logger).Log("delete object", name)

	return s.conn.ObjectDelete(s.getBucketName(name), name)
}

func (s *SwiftStorage) Get(name string, output io.Writer) (err error) {
	//noinspection GoUnhandledErrorResult
	level.Debug(s.logger).Log("get object", name)

	_, err = s.conn.ObjectGet(s.getBucketName(name), name, output, true, swift.Headers{})
	return
}

func (s *SwiftStorage) PruneOldest(date time.Time) error {
	removeCh := make(chan swift.Object, 10)
	defer close(removeCh)

	for i := 0; i < 10; i++ {
		//noinspection GoUnhandledErrorResult
		level.Debug(s.logger).Log("run prune worker #", i)

		go func(i int) {
			for object := range removeCh {
				if object.LastModified.Before(date) {
					//noinspection GoUnhandledErrorResult
					level.Info(s.logger).Log("worker", i, "remove file", object.Name)
					if err := s.Delete(object.Name); err != nil {
						//noinspection GoUnhandledErrorResult
						level.Error(s.logger).Log("worker", i, "cannot remove object", object.Name, err)
					}
				}
			}
		}(i)
	}

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
