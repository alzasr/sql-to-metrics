package internal

import "time"

type Settings struct {
	Host string       `yaml:"host"`
	Path string       `yaml:"path"`
	Jobs []MetricsJob `yaml:"jobs"`
}

type MetricsJob struct {
	DB      DBSettings    `yaml:"db"`
	Period  time.Duration `yaml:"period"`
	Metrics []Metric      `yaml:"metrics"`
}

type MetricType = string

const (
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

type Metric struct {
	Name   string     `yaml:"name"`
	Query  string     `yaml:"query"`
	Type   MetricType `yaml:"type"`
	Values []string   `yaml:"values"`
}

type DBSettings struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type metricExecutor func() error
