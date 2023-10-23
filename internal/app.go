package internal

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
	metricExecutors := make([]metricExecutor, 0, len(job.Metrics))

	for _, metric := range job.Metrics {
		executor, err := makeMetricExecutor(registry, db, metric)
		if err != nil {
			return fmt.Errorf("make metric executor (name=%s)", metric.Name)
		}
		metricExecutors = append(metricExecutors, executor)
	}

	for _, executor := range metricExecutors {
		execErr := executor()
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
				err := executor()
				if err != nil {
					slog.Error(err.Error())
				}
			}
		}
	}()
	return nil
}

func makeMetricExecutor(registry *prometheus.Registry, db *sql.DB, metric Metric) (metricExecutor, error) {
	switch metric.Type {
	case MetricTypeGauge:
		return makeMetricExecutorGauge(registry, db, metric)
	}
	return nil, fmt.Errorf("unsupported metric type %s", metric.Type)
}

func makeMetricExecutorGauge(registry *prometheus.Registry, db *sql.DB, metric Metric) (metricExecutor, error) {
	gauges := make(map[string]prometheus.Gauge, len(metric.Values))
	for _, value := range metric.Values {
		gauges[value] = promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: metric.Name,
			ConstLabels: map[string]string{
				"valueField": value,
			},
		})
	}
	return func() error {
		res, err := db.Query(metric.Query)
		if err != nil {
			return fmt.Errorf("query: %w", err)
		}
		defer func() {
			_ = res.Close()
		}()
		colums, err := res.Columns()
		if err != nil {
			return fmt.Errorf("colums: %w", err)
		}

		has := res.Next()
		if !has {
			return fmt.Errorf("result is empty")
		}
		values := make(map[string]*float64, len(colums))
		anyValues := make([]any, len(colums))
		for i := range colums {
			v := 0.0
			values[colums[i]] = &v
			anyValues[i] = &v
		}
		err = res.Scan(anyValues...)
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		for _, value := range metric.Values {
			gauges[value].Set(*values[value])
		}
		return nil
	}, nil
}
