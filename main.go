package main

import (
	"database/sql"
	"net/http"
	"os"

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
	log.Printf("INFO: Starting http server - %s", "0.0.0.0:9560")
	if err := http.ListenAndServe("0.0.0.0:9560", nil); err != nil {
		log.Printf("ERROR: Failed to start http server: %s", err)
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

func gethostname() string {
	host, err := os.Hostname()
	if err != nil {
		log.Fatalf("ERROR: Hostname fetching err", err)
	}
	return host
}
func configConstructer() Config {
	return Config{
		URI:    "mysqld_exporter:StrongPassword@tcp(127.0.0.1:3306)/mysql",
		Query:  "SELECT sum(data_length + index_length)/1024/1024 as document_count FROM information_schema.TABLES;",
		Type:   "Gauge",
		Value:  "document_count",
		Labels: []string{"instance"},
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
	log.Printf("INFO: metric description for \"%s\" registerd", "mariadb_exporter")
}

// Collect -> prometheus collect
func (e *QueryCollector) Collect(ch chan<- prometheus.Metric) {
	db, err := sql.Open("mysql", cfg.URI)

	if err != nil {
		log.Printf("ERROR: Connect to database failed: %s", err)
		return
	}
	defer db.Close()

	rows := db.QueryRow(cfg.Query).Scan(&Result)
	if rows != nil {
		log.Printf("ERROR: Querying failed", err)
	}

	// Metric labels - Not doing anything here but might be useful
	labelVals := []string{}
	for range cfg.Labels {
		labelVals = append(labelVals, gethostname())
	}
	// Run Collector channel
	ch <- prometheus.MustNewConstMetric(cfg.metricDesc, prometheus.GaugeValue, Result, labelVals...)
}
