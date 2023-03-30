/*****************************************************************************
*
*	File			: main.go
*
* 	Created			: 27 March 2023
*
*	Description		: Quick Dirty wrapper for Prometheus and golang library to figure out how to back port it into fs_loader
*
*	Modified		: 27 March 2023	- Start
*
*	By			: George Leonard (georgelza@gmail.com)
*
*
*
*****************************************************************************/

package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metrics struct {
	info          *prometheus.GaugeVec
	req_processed *prometheus.CounterVec
	sql_duration  *prometheus.HistogramVec
	rec_duration  *prometheus.HistogramVec
	api_duration  *prometheus.HistogramVec
}

var (
	reg = prometheus.NewRegistry()
	m   = NewMetrics(reg)
)

func NewMetrics(reg prometheus.Registerer) *metrics {

	m := &metrics{
		info: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "txn_count",
			Help: "Target amount for completed requests",
		}, []string{"batch"}),

		req_processed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lb_fs_etl_operations_count",
			Help: "Number of completed requests.",
		}, []string{"batch"}),

		sql_duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "lb_fs_sql_duration_seconds",
			Help: "Duration of the sql requests",
			// 4 times larger apdex status
			// Buckets: prometheus.ExponentialBuckets(0.1, 1.5, 5),
			// Buckets: prometheus.LinearBuckets(0.1, 5, 5),
			Buckets: []float64{0.1, 0.15, 0.2, 0.25, 0.3},
		}, []string{"batch"}),

		rec_duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "lb_fs_etl_operations_seconds",
			Help: "Duration of the entire requests",

			Buckets: []float64{0.1, 0.15, 0.2, 0.25, 0.3},
		}, []string{"batch"}),

		api_duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "lb_fs_api_duration_seconds",
			Help:    "Duration of the api requests",
			Buckets: []float64{0.1, 0.15, 0.2, 0.25, 0.3},
		}, []string{"batch"}),
	}

	reg.MustRegister(m.info, m.req_processed, m.sql_duration, m.rec_duration, m.api_duration)

	return m
}

// I'm showing how both the eft and acc can be pointed to record their metrics to the same prometheus counters,  using differentiated "WithLabelValues" as
// available when athe metric is defined as prometheus.*Vec (vector based) metric .
func loadEFT() {

	var x int = 200
	var txn_count float64

	////////////////////////////////
	// start a timer
	sTime := time.Now()

	txn_count = 9752395 // this will be the recordcount of the records returned by the sql query
	m.info.With(prometheus.Labels{"batch": "eft"}).Set(txn_count)

	// Execute a large sql #1 execute
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(10000) // if vGeneral.sleep = 10000, 10 second
	fmt.Printf("EFT SQL Sleeping %d Millisecond...\n", n)
	time.Sleep(time.Duration(n) * time.Millisecond)

	// post to Prometheus
	sqlDuration := time.Since(sTime)
	m.sql_duration.WithLabelValues("nrt_eft").Observe(sqlDuration.Seconds())

	////////////////////////////////
	// restart timer
	sTime = time.Now()

	txn_count = 104565
	m.info.With(prometheus.Labels{"batch": "acc"}).Set(txn_count)

	// Execute a large sql #2 execute
	rand.Seed(time.Now().UnixNano())
	n = rand.Intn(10000) // if vGeneral.sleep = 10000, 10 second
	fmt.Printf("ACC SQL Sleeping %d Millisecond...\n", n)
	time.Sleep(time.Duration(n) * time.Millisecond)

	// post to Prometheus
	sqlDuration = time.Since(sTime)
	m.sql_duration.WithLabelValues("nrt_acc").Observe(sqlDuration.Seconds())

	for count := 0; count < x; count++ {

		sT := time.Now()

		// EFT
		sTime = time.Now()
		rand.Seed(time.Now().UnixNano())
		n := rand.Intn(5000) // if vGeneral.sleep = 1000, then n will be random value of 0 -> 1000  aka 0 and 1 second
		fmt.Printf("Sleeping %d Millisecond...\n", n)
		time.Sleep(time.Duration(n) * time.Millisecond)

		//determine the duration and log to prometheus
		fsDuration := time.Since(sTime)
		m.api_duration.WithLabelValues("nrt_eft").Observe(fsDuration.Seconds())
		//increment a counter for number of requests processed
		m.req_processed.WithLabelValues("nrt_eft").Inc()

		// ACC
		sTime = time.Now()
		rand.Seed(time.Now().UnixNano())
		n = rand.Intn(5000) // if vGeneral.sleep = 1000, then n will be random value of 0 -> 1000  aka 0 and 1 second
		fmt.Printf("Sleeping %d Millisecond...\n", n)
		time.Sleep(time.Duration(n) * time.Millisecond)

		//determine the duration and log to prometheus
		fsDuration = time.Since(sTime)
		m.api_duration.WithLabelValues("nrt_acc").Observe(fsDuration.Seconds())
		//increment a counter for number of requests processed
		m.req_processed.WithLabelValues("nrt_acc").Inc()

		//determine the duration and log to prometheus
		fsD := time.Since(sT)
		m.rec_duration.WithLabelValues("nrt_eft").Observe(fsD.Seconds())
		m.rec_duration.WithLabelValues("nrt_acc").Observe(fsD.Seconds())

		println(count)
	}
	os.Exit(0)
}

func main() {

	pMux := http.NewServeMux()
	promHandler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	pMux.Handle("/metrics", promHandler)

	fmt.Println("Starting...")

	go loadEFT()

	go func() {
		fmt.Println(http.ListenAndServe(":9000", pMux))
	}()
	select {}

}
