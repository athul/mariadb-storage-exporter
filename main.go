package main

import (
	"database/sql"
	"net/http"

	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
)

var cfg Config

func main() {
	cfg = configConstructer()
	prometheus.Register(version.NewCollector("mariadb_exporter"))
	prometheus.Register(&QueryCollector{})
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		h := promhttp.HandlerFor(prometheus.Gatherers{
			prometheus.DefaultGatherer,
		}, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	})
	log.Println("Starting http server - %s", "0.0.0.0:9560")
	if err := http.ListenAndServe("0.0.0.0:9560", nil); err != nil {
		log.Printf("Failed to start http server: %s", err)
	}
}

type Config struct {
	URI        string
	Query      string
	Type       string
	Value      string
	Labels     []string
	metricDesc *prometheus.Desc
}

var Result float64

func configConstructer() Config {
	return Config{
		URI:    "mysqld_exporter:StrongPassword@tcp(127.0.0.1:3306)/mysql",
		Query:  "SELECT sum(data_length + index_length)/1024/1024 as document_count FROM information_schema.TABLES;",
		Type:   "Gauge",
		Value:  "document_count",
		Labels: []string{"mariadb"},
	}
}

type QueryCollector struct{}

// Describe -> prometheus describe
func (e *QueryCollector) Describe(ch chan<- *prometheus.Desc) {
	cfg.metricDesc = prometheus.NewDesc(
		prometheus.BuildFQName("mariadb_exporter", "", "mariadb_storage_usage"),
		"Exporter for Mariadb Storage Usage in MB",
		cfg.Labels, nil,
	)
	log.Println("metric description for \"%s\" registerd", "mariadb_exporter")
}

// Collect -> prometheus collect
func (e *QueryCollector) Collect(ch chan<- prometheus.Metric) {
	db, err := sql.Open("mysql", cfg.URI)

	if err != nil {
		log.Printf("Connect to database failed: %s", err)
		return
	}
	defer db.Close()

	rows := db.QueryRow(cfg.Query).Scan(&Result)
	if rows != nil {
		log.Printf("Querying failed", err)
	}

	// Metric labels - Not doing anything here but might be useful
	data := make(map[string]string)
	labelVals := []string{}
	for _, label := range cfg.Labels {
		labelVals = append(labelVals, data[label])
	}
	// Add metric
	ch <- prometheus.MustNewConstMetric(cfg.metricDesc, prometheus.GaugeValue, Result, cfg.Labels...)
}
