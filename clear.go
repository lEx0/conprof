package main

import (
	"github.com/go-kit/kit/log"
	"github.com/lEx0/conprof/rtsb/storage"
	"github.com/oklog/run"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"time"
)

func RegisterClear(m map[string]setupFunc, app *kingpin.Application, name string) {
	cmd := app.Command(name, "clear oldest entries from remote storage")
	retention := modelDuration(cmd.Flag(
		"storage.tsdb.retention.time",
		"How long to retain raw samples on local storage. 0d - disables this retention",
	).Default("15d"))
	remoteStorageUrl := cmd.Flag(
		"storage.tsdb.remote.url",
		"Binary profiles storage URL",
	).Required().String()

	m[name] = func(
		_ *run.Group,
		_ *http.ServeMux,
		logger log.Logger,
		_ *prometheus.Registry,
		_ opentracing.Tracer,
		_ bool,
	) error {
		if err := runClear(logger, *remoteStorageUrl, *retention); err != nil {
			//noinspection GoUnhandledErrorResult
			logger.Log("run clear is failed", err)
			os.Exit(1)
		}
		os.Exit(0)
		return nil
	}
}

func runClear(logger log.Logger, url string, retention model.Duration) (err error) {
	var strg storage.Storage

	if strg, err = storage.NewSwiftStorage(storage.Options{
		URL:     url,
		Timeout: time.Second * 10,
		Bucket:  "pprof",
	}, logger); err != nil {
		//noinspection GoUnhandledErrorResult
		logger.Log("cannot connect to storage", err)
		return
	}

	return strg.PruneOldest(time.Now().Add(-time.Duration(retention)))
}
