package collector

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type dladmLinkConfigOutput struct {
	link   string
	class  string
	mtu    uint64
	state  string
	bridge string
	over   string
}

type dladmLinkStatsOutput struct {
	link     string
	iPackets uint64
	oPackets uint64
	rBytes   uint64
	oBytes   uint64
	iErrors  uint64
	oErrors  uint64
}

type netCollector struct {
	iPackets *prometheus.GaugeVec
	oPackets *prometheus.GaugeVec
	rBytes   *prometheus.GaugeVec
	oBytes   *prometheus.GaugeVec
	iErrors  *prometheus.GaugeVec
	oErrors  *prometheus.GaugeVec
	class    *prometheus.GaugeVec
	mtu      *prometheus.GaugeVec
	state    *prometheus.GaugeVec
	bridge   *prometheus.GaugeVec
	over     *prometheus.GaugeVec
}

const (
	netCollectorSubsystem = "net_link"
)

func init() {
	registerCollector(netCollectorSubsystem, defaultEnabled, NewNetCollector)
}

func NewNetCollector(logger log.Logger) (Collector, error) {
	return &netCollector{
		iPackets: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "ipackets"),
			Help: "Link input packets",
		}, []string{"link", "timestamp"}),
		oPackets: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "opackets"),
			Help: "Link output packets",
		}, []string{"link", "timestamp"}),
		rBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "rbytes"),
			Help: "Link received bytes",
		}, []string{"link", "timestamp"}),
		oBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "obytes"),
			Help: "Link transmitted bytes",
		}, []string{"link", "timestamp"}),
		iErrors: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "ierrors"),
			Help: "Link receive errors",
		}, []string{"link", "timestamp"}),
		oErrors: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "oerrors"),
			Help: "Link output errors",
		}, []string{"link", "timestamp"}),
		class: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "class"),
			Help: "Link class",
		}, []string{"link", "class", "timestamp"}),
		mtu: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "mtu"),
			Help: "Link MTU",
		}, []string{"link", "mtu", "timestamp"}),
		state: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "state"),
			Help: "Link state",
		}, []string{"link", "state", "timestamp"}),
		bridge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "bridge"),
			Help: "Link bridge",
		}, []string{"link", "bridge", "timestamp"}),
		over: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "over"),
			Help: "Link over",
		}, []string{"link", "over", "timestamp"}),
	}, nil
}

func (c *netCollector) dladmConfGet() error {
	configs, err := getDladmConfig()
	if err != nil {
		return fmt.Errorf("getting dladm links configs: %w", err)
	}

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	for _, conf := range configs {
		mtu := strconv.FormatUint(conf.mtu, 10)

		c.class.With(
			prometheus.Labels{"link": conf.link, "class": conf.class, "timestamp": timestamp},
		).Set(0)
		c.mtu.With(
			prometheus.Labels{"link": conf.link, "mtu": mtu, "timestamp": timestamp},
		).Set(0)
		c.state.With(
			prometheus.Labels{"link": conf.link, "state": conf.state, "timestamp": timestamp},
		).Set(0)
		c.bridge.With(
			prometheus.Labels{"link": conf.link, "bridge": conf.bridge, "timestamp": timestamp},
		).Set(0)
		c.over.With(
			prometheus.Labels{"link": conf.link, "over": conf.over, "timestamp": timestamp},
		).Set(0)
	}
	return nil
}

func (c *netCollector) dladmStatsGet() error {
	linksStats, err := getDladmStats()
	if err != nil {
		return fmt.Errorf("getting dladm links stats: %w", err)
	}

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	for _, stats := range linksStats {
		c.iPackets.With(
			prometheus.Labels{"link": stats.link, "timestamp": timestamp},
		).Set(float64(stats.iPackets))
		c.oPackets.With(
			prometheus.Labels{"link": stats.link, "timestamp": timestamp},
		).Set(float64(stats.oPackets))
		c.rBytes.With(
			prometheus.Labels{"link": stats.link, "timestamp": timestamp},
		).Set(float64(stats.rBytes))
		c.oBytes.With(
			prometheus.Labels{"link": stats.link, "timestamp": timestamp},
		).Set(float64(stats.oBytes))
		c.iErrors.With(
			prometheus.Labels{"link": stats.link, "timestamp": timestamp},
		).Set(float64(stats.iErrors))
		c.oErrors.With(
			prometheus.Labels{"link": stats.link, "timestamp": timestamp},
		).Set(float64(stats.oErrors))
	}
	return nil
}

func (c *netCollector) Update(ch chan<- prometheus.Metric) error {
	c.dladmConfGet()
	c.dladmStatsGet()
	c.iPackets.Collect(ch)
	c.oPackets.Collect(ch)
	c.rBytes.Collect(ch)
	c.oBytes.Collect(ch)
	c.iErrors.Collect(ch)
	c.oErrors.Collect(ch)
	c.class.Collect(ch)
	c.mtu.Collect(ch)
	c.state.Collect(ch)
	c.bridge.Collect(ch)
	c.over.Collect(ch)
	return nil
}

func (c *netCollector) Describe(ch chan<- *prometheus.Desc) {
	c.iPackets.Describe(ch)
	c.oPackets.Describe(ch)
	c.rBytes.Describe(ch)
	c.oBytes.Describe(ch)
	c.iErrors.Describe(ch)
	c.oErrors.Describe(ch)
	c.class.Describe(ch)
	c.mtu.Describe(ch)
	c.state.Describe(ch)
	c.bridge.Describe(ch)
	c.over.Describe(ch)
}

func getDladmConfig() ([]dladmLinkConfigOutput, error) {
	out, err := exec.Command(
		"dladm", "show-link", "-po",
		"link,class,mtu,state,bridge,over",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("dladm: %w", err)
	}

	var configs []dladmLinkConfigOutput

	reader := bytes.NewReader(out)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		values := strings.Split(scanner.Text(), ":")
		if values[0] == "" {
			continue
		}

		mtu, err := strconv.ParseUint(values[2], 10, 64)
		if err != nil {
			return nil, newDladmParsingError(values[0], "mtu", err)
		}

		conf := dladmLinkConfigOutput{
			link:   values[0],
			class:  values[1],
			mtu:    mtu,
			state:  values[3],
			bridge: values[4],
			over:   values[5],
		}
		configs = append(configs, conf)
	}

	if scanner.Err() != nil {
		return nil, fmt.Errorf("reading dladm config output: %w", err)
	}
	return configs, nil
}

func getDladmStats() ([]dladmLinkStatsOutput, error) {
	out, err := exec.Command(
		"dladm", "show-link", "-pso",
		"link,ipackets,opackets,rbytes,obytes,ierrors,oerrors",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("dladm: %w", err)
	}

	var linksStats []dladmLinkStatsOutput

	reader := bytes.NewReader(out)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		values := strings.Split(scanner.Text(), ":")
		if values[0] == "" {
			continue
		}
		link := values[0]
		iPackets, err := strconv.ParseUint(values[1], 10, 64)
		if err != nil {
			return nil, newDladmParsingError(link, "ipackets", err)
		}
		oPackets, err := strconv.ParseUint(values[2], 10, 64)
		if err != nil {
			return nil, newDladmParsingError(link, "opackets", err)
		}
		rBytes, err := strconv.ParseUint(values[3], 10, 64)
		if err != nil {
			return nil, newDladmParsingError(link, "rbytes", err)
		}
		oBytes, err := strconv.ParseUint(values[4], 10, 64)
		if err != nil {
			return nil, newDladmParsingError(link, "obytes", err)
		}
		iErrors, err := strconv.ParseUint(values[5], 10, 64)
		if err != nil {
			return nil, newDladmParsingError(link, "ierrors", err)
		}
		oErrors, err := strconv.ParseUint(values[6], 10, 64)
		if err != nil {
			return nil, newDladmParsingError(link, "oerrors", err)
		}

		stats := dladmLinkStatsOutput{
			link:     link,
			iPackets: iPackets,
			oPackets: oPackets,
			rBytes:   rBytes,
			oBytes:   oBytes,
			iErrors:  iErrors,
			oErrors:  oErrors,
		}
		linksStats = append(linksStats, stats)
	}

	if scanner.Err() != nil {
		return nil, fmt.Errorf("reading dladm stats output: %w", err)
	}
	return linksStats, nil
}

func newDladmParsingError(link, field string, err error) error {
	return fmt.Errorf("parsing '%s' for link '%s': %w", field, link, err)
}
