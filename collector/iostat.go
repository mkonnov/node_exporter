// iostat collector
// this will :
//  - call iostat
//  - gather hard disk metrics
//  - feed the collector

package collector

import (
	"github.com/prometheus/node_exporter/collector/metric"
	_ "net/http/pprof"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerCollector("iostat", defaultEnabled, NewGZDiskErrorsExporter)
}

// GZDiskErrorsCollector declares the data type within the prometheus metrics package.
type GZDiskErrorsCollector struct {
	gzDiskErrors *prometheus.GaugeVec
	logger       log.Logger
}

// NewGZDiskErrorsExporter returns a newly allocated exporter GZDiskErrorsCollector.
// It exposes the number of hardware disk errors
func NewGZDiskErrorsExporter(logger log.Logger) (Collector, error) {

	return &GZDiskErrorsCollector{
		gzDiskErrors: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "node_iostat_disk_errs_total",
			Help: "Number of hardware disk errors.",
		}, metric.NewLabelNames("device", "error_type")),
		logger: logger,
	}, nil
}

// Describe describes all the metrics.
func (e *GZDiskErrorsCollector) Describe(ch chan<- *prometheus.Desc) {
	e.gzDiskErrors.Describe(ch)
}

// Collect fetches the stats.
func (e *GZDiskErrorsCollector) Update(ch chan<- prometheus.Metric) error {
	e.iostat()
	e.gzDiskErrors.Collect(ch)
	return nil
}

func (e *GZDiskErrorsCollector) iostat() {
	out, eerr := exec.Command("iostat", "-en").Output()
	if eerr != nil {
		level.Error(e.logger).Log("error on executing iostat: %v", eerr)
	}
	perr := e.parseIostatOutput(string(out))
	if perr != nil {
		level.Error(e.logger).Log("error on parsing iostat: %v", perr)
	}
}

func (e *GZDiskErrorsCollector) parseIostatOutput(out string) error {
	outlines := strings.Split(out, "\n")
	l := len(outlines)
	for _, line := range outlines[2 : l-1] {
		parsedLine := strings.Fields(line)
		deviceName := parsedLine[4]
		softErr, err := strconv.ParseFloat(parsedLine[0], 64)
		if err != nil {
			return err
		}
		hardErr, err := strconv.ParseFloat(parsedLine[1], 64)
		if err != nil {
			return err
		}
		trnErr, err := strconv.ParseFloat(parsedLine[2], 64)
		if err != nil {
			return err
		}
		e.gzDiskErrors.With(
			metric.NewLabels(map[string]string{"device": deviceName, "error_type": "soft"}),
		).Set(softErr)
		e.gzDiskErrors.With(
			metric.NewLabels(map[string]string{"device": deviceName, "error_type": "hard"}),
		).Set(hardErr)
		e.gzDiskErrors.With(
			metric.NewLabels(map[string]string{"device": deviceName, "error_type": "trn"}),
		).Set(trnErr)
	}
	return nil
}
