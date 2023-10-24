package internal

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"log/slog"
	"net/http"
	"time"
)

func Run(settings *Settings) (err error) {
	registry := prometheus.NewRegistry()

	for _, job := range settings.Jobs {
		err = startJob(registry, job)
		if err != nil {
			log.Fatal(err)
		}
	}

	http.Handle(settings.Path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	err = http.ListenAndServe(settings.Host, nil)
	if err != nil {
		log.Panic(err)
	}
	return nil
}

func startJob(registry *prometheus.Registry, job MetricsJob) error {
	db, err := sql.Open(job.DB.Driver, job.DB.DSN)
	if err != nil {
		return fmt.Errorf("sql open: %w", err)
	}
	metricExecutors := make([]MetricExecutor, 0, len(job.Metrics))

	for i := range job.Metrics {
		executor, err := buildExecutor(registry, db, &job.Metrics[i])
		if err != nil {
			return fmt.Errorf("make metric executor (name=%s)", job.Metrics[i].Name)
		}
		metricExecutors = append(metricExecutors, executor)
	}

	for _, executor := range metricExecutors {
		execErr := executor.Run()
		if execErr != nil {
			err = errors.Join(err, execErr)
		}
	}

	if err != nil {
		return fmt.Errorf("fail test executors: %w", err)
	}

	go func() {
		for {
			time.Sleep(job.Period)
			for _, executor := range metricExecutors {
				err := executor.Run()
				if err != nil {
					slog.Error(err.Error())
				}
			}
		}
	}()
	return nil
}
