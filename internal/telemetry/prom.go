package telemetry

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metrics struct {
	requestTime      *prometheus.HistogramVec
	containerRequest *prometheus.CounterVec
	frontendActive   *prometheus.GaugeVec
	cacheSize        prometheus.Gauge
	cacheLength      prometheus.Gauge
	containerRPS     *prometheus.GaugeVec
}

var (
	reg *prometheus.Registry
	m   *metrics
)

func NewMetrics(reg prometheus.Registerer) *metrics {
	histogramLabels := []string{"method", "container", "path", "status"}
	histogramBuckets := []float64{0, 1, 10, 100, 300, 500, 1000, 5000}
	m := &metrics{
		requestTime: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "request_time",
			Help:    "Request time in ms",
			Buckets: histogramBuckets,
		}, histogramLabels),
		containerRequest: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "container_request",
			Help: "Container requests",
		}, []string{"container", "error"}),
		frontendActive: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "frontend_active",
			Help: "Frontend active count",
		}, []string{"remote_addr"}),
		cacheSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "cache_size",
			Help: "Cache size in items",
		}),
		cacheLength: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "cache_length",
			Help: "Cache length in bytes",
		}),
		containerRPS: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "container_rps",
			Help: "Container limit request per second",
		}, []string{"container"}),
	}

	reg.MustRegister(m.requestTime, m.containerRequest, m.frontendActive, m.cacheLength, m.cacheSize, m.containerRPS)
	return m
}

func PromHandler() http.Handler {

	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg})
}

func ReqTime(r *http.Request, rt time.Duration, status int) {
	m.requestTime.WithLabelValues(r.Method, GetContainer(r), r.URL.Path, strconv.Itoa(status)).Observe(float64(rt.Milliseconds()))
	if status > 0 {
		m.containerRequest.WithLabelValues(GetContainer(r), "").Inc()
	}
}

func RateLimit(r *http.Request) {
	m.containerRequest.WithLabelValues(GetContainer(r))
}

func RequestError(r *http.Request, err error) {
	m.containerRequest.WithLabelValues(GetContainer(r), err.Error()).Inc()
}

func EnableFrontend(r *http.Request) {
	m.frontendActive.WithLabelValues(r.RemoteAddr).Inc()
}

func DisableFrontend(r *http.Request) {
	m.frontendActive.WithLabelValues(r.RemoteAddr).Dec()
}

func SetCacheLength(l int) {
	m.cacheSize.Set(float64(l))
}

func SetCacheSize(cacheSize uint64) {
	m.cacheLength.Set(float64(cacheSize))
}

func SetContainerRPS(container string, rps int) {
	m.containerRPS.WithLabelValues(container).Set(float64(rps))
}

func init() {
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	m = NewMetrics(reg)
}
