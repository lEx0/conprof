// Copyright 2018 The conprof Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"net/http"
	"time"

	"github.com/conprof/tsdb"
	"github.com/conprof/tsdb/wal"
	"github.com/go-kit/kit/log"
	"github.com/lEx0/conprof/rtsb"
	"github.com/lEx0/conprof/rtsb/storage"
	"github.com/oklog/run"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// registerAll registers the all command.
func registerAll(m map[string]setupFunc, app *kingpin.Application, name string) {
	cmd := app.Command(name, "All in one command.")

	storagePath := cmd.Flag("storage.tsdb.path", "Directory to read storage from.").
		Default("./data").String()
	configFile := cmd.Flag("config.file", "Config file to use.").
		Default("conprof.yaml").String()
	retention := modelDuration(cmd.Flag(
		"storage.tsdb.retention.time",
		"How long to retain raw samples on local storage. 0d - disables this retention",
	).Default("15d"))
	remoteStorageUrl := cmd.Flag("storage.tsdb.remote.url", "Binary profiles storage URL").
		Default("").String()
	remoteStoragePartitionKeySize := cmd.Flag("storage.tsdb.remote.patition", "size of partition key").
		Default("0").Int()

	m[name] = func(
		g *run.Group,
		mux *http.ServeMux,
		logger log.Logger,
		reg *prometheus.Registry,
		tracer opentracing.Tracer,
		debugLogging bool,
	) error {
		return runAll(g, mux, logger, *storagePath, *configFile, *remoteStorageUrl, *retention, *remoteStoragePartitionKeySize)
	}
}

func modelDuration(flags *kingpin.FlagClause) *model.Duration {
	var value = new(model.Duration)
	flags.SetValue(value)

	return value
}

func runAll(
	g *run.Group,
	mux *http.ServeMux,
	logger log.Logger,
	storagePath,
	configFile,
	remoteStorageUrl string,
	retention model.Duration,
	partitionKeySize int,
) error {
	db, err := tsdb.Open(
		storagePath,
		logger,
		prometheus.DefaultRegisterer,
		&tsdb.Options{
			WALSegmentSize:    wal.DefaultSegmentSize,
			RetentionDuration: uint64(retention),
			BlockRanges:       tsdb.ExponentialBlockRanges(int64(2*time.Hour)/1e6, 3, 5),
			NoLockfile:        true,
		},
	)
	if err != nil {
		return err
	}

	var strg storage.Storage

	if remoteStorageUrl != "" {
		if strg, err = storage.NewSwiftStorage(storage.Options{
			URL:                    remoteStorageUrl,
			Timeout:                time.Second * 10,
			Bucket:                 "pprof",
			PartitionPrefixKeySize: partitionKeySize,
		}, logger); err != nil {
			return err
		} else if err = runSampler(g, logger, rtsb.NewRemoteTSDB(strg, db), configFile); err != nil {
			return err
		}
	} else if err = runSampler(g, logger, db, configFile); err != nil {
		return err
	}

	err = runWeb(mux, logger, db, strg)
	if err != nil {
		return err
	}

	return nil
}
