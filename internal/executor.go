package internal

import (
	"database/sql"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"log/slog"
	"strconv"
	"sync"
)

func buildExecutor(registry *prometheus.Registry, db *sql.DB, metric *Metric) (res MetricExecutor, err error) {
	switch metric.Type {
	case MetricTypeGauge:
		res, err = newGaugeMetric(registry, db, metric)
	default:
		return nil, fmt.Errorf("unsupported metric type %s", metric.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("make executor: %w", err)
	}
	return res, err
}

var _ MetricExecutor = &gaugeMetric{}

func newGaugeMetric(registry *prometheus.Registry, db *sql.DB, metric *Metric) (MetricExecutor, error) {
	return &gaugeMetric{
		db,
		registry,
		metric.Name,
		metric.Query,
		metric.Values,
		metric.Labels,
		map[string]map[string]prometheus.Gauge{},
		sync.Mutex{},
	}, nil
}

type gaugeMetric struct {
	db           *sql.DB
	registry     *prometheus.Registry
	name         string
	query        string
	valuesFields []string
	labelsFields []string
	collectors   map[string]map[string]prometheus.Gauge
	mu           sync.Mutex
}

func (g *gaugeMetric) Run() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	res, err := g.db.Query(g.query)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer func() {
		_ = res.Close()
	}()
	columns, err := res.Columns()
	if err != nil {
		return fmt.Errorf("colums: %w", err)
	}

	for res.Next() {
		values := make(map[string]*string, len(columns))
		anyValues := make([]any, len(columns))
		for i := range columns {
			v := ""
			values[columns[i]] = &v
			anyValues[i] = &v
		}
		err = res.Scan(anyValues...)
		labels := fromRes(values, g.labelsFields)
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		for _, field := range g.valuesFields {
			res, err := convertValue(values[field])
			if err != nil {
				slog.Error(fmt.Errorf("convert value: %w", err).Error(), "field", field)
			}
			g.getCollector(field, labels).Set(res)
		}
	}
	return nil
}

func convertValue(str *string) (float64, error) {
	if str == nil || *str == "" {
		return 0.0, nil
	}
	return strconv.ParseFloat(*str, 64)
}

func (g *gaugeMetric) getCollector(field string, labels Labels) prometheus.Gauge {

	hash := labels.String()

	if _, ok := g.collectors[field]; !ok {
		g.collectors[field] = map[string]prometheus.Gauge{}
	}
	if _, ok := g.collectors[field][hash]; !ok {
		collectorLabels := make(Labels, len(labels)+1)
		for k, v := range labels {
			collectorLabels[k] = v
		}
		collectorLabels["valueField"] = field
		g.collectors[field][hash] = promauto.With(g.registry).NewGauge(prometheus.GaugeOpts{
			Name:        g.name,
			ConstLabels: prometheus.Labels(collectorLabels),
		})
	}
	return g.collectors[field][hash]
}
