package collector

import (
	"fmt"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/node_exporter/collector/metric"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

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
		}, metric.NewLabelNames("link", "timestamp")),
		oPackets: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "opackets"),
			Help: "Link output packets",
		}, metric.NewLabelNames("link", "timestamp")),
		rBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "rbytes"),
			Help: "Link received bytes",
		}, metric.NewLabelNames("link", "timestamp")),
		oBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "obytes"),
			Help: "Link transmitted bytes",
		}, metric.NewLabelNames("link", "timestamp")),
		iErrors: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "ierrors"),
			Help: "Link receive errors",
		}, metric.NewLabelNames("link", "timestamp")),
		oErrors: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "oerrors"),
			Help: "Link output errors",
		}, metric.NewLabelNames("link", "timestamp")),
		class: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "class"),
			Help: "Link class",
		}, metric.NewLabelNames("link", "class", "timestamp")),
		mtu: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "mtu"),
			Help: "Link MTU",
		}, metric.NewLabelNames("link", "mtu", "timestamp")),
		state: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "state"),
			Help: "Link state",
		}, metric.NewLabelNames("link", "state", "timestamp")),
		bridge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "bridge"),
			Help: "Link bridge",
		}, metric.NewLabelNames("link", "bridge", "timestamp")),
		over: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, netCollectorSubsystem, "over"),
			Help: "Link over",
		}, metric.NewLabelNames("link", "over", "timestamp")),
	}, nil
}

func (c *netCollector) dladmConfGet() error {
	var (
		link, class, mtu, state, bridge, over string
		err                                   error
	)
	out, err := exec.Command("dladm", "show-link", "-po",
		"link,class,mtu,state,bridge,over").Output()
	if err != nil {
		return err
	}

	outlines := strings.Split(string(out), "\n")
	for _, l := range outlines {
		values := strings.Split(l, ":")
		if values[0] == "" {
			continue
		}
		link = values[0]
		class = values[1]
		mtu = values[2]
		state = values[3]
		bridge = values[4]
		over = values[5]

		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

		c.class.With(
			metric.NewLabels(map[string]string{"link": link, "class": class, "timestamp": timestamp}),
		).Set(0)
		c.mtu.With(
			metric.NewLabels(map[string]string{"link": link, "mtu": mtu, "timestamp": timestamp}),
		).Set(0)
		c.state.With(
			metric.NewLabels(map[string]string{"link": link, "state": state, "timestamp": timestamp}),
		).Set(0)
		c.bridge.With(
			metric.NewLabels(map[string]string{"link": link, "bridge": bridge, "timestamp": timestamp}),
		).Set(0)
		c.over.With(
			metric.NewLabels(map[string]string{"link": link, "over": over, "timestamp": timestamp}),
		).Set(0)
	}
	return nil
}

func (c *netCollector) dladmStatsGet() error {
	var (
		link string
		ipackets, opackets, rbytes,
		obytes, ierrors, oerrors uint64
		err error
	)
	out, err := exec.Command("dladm", "show-link", "-pso",
		"link,ipackets,opackets,rbytes,obytes,ierrors,oerrors").Output()
	if err != nil {
		return err
	}

	outlines := strings.Split(string(out), "\n")
	for _, l := range outlines {
		values := strings.Split(l, ":")
		if values[0] == "" {
			continue
		}
		link = values[0]
		ipackets, err = strconv.ParseUint(values[1], 10, 64)
		if err != nil {
			return err
		}
		opackets, err = strconv.ParseUint(values[2], 10, 64)
		if err != nil {
			return err
		}
		rbytes, err = strconv.ParseUint(values[3], 10, 64)
		if err != nil {
			return err
		}
		obytes, err = strconv.ParseUint(values[4], 10, 64)
		if err != nil {
			return err
		}
		ierrors, err = strconv.ParseUint(values[5], 10, 64)
		if err != nil {
			return err
		}
		oerrors, err = strconv.ParseUint(values[6], 10, 64)
		if err != nil {
			return err
		}

		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

		c.iPackets.With(
			metric.NewLabels(map[string]string{"link": link, "timestamp": timestamp}),
		).Set(float64(ipackets))
		c.oPackets.With(
			metric.NewLabels(map[string]string{"link": link, "timestamp": timestamp}),
		).Set(float64(opackets))
		c.rBytes.With(
			metric.NewLabels(map[string]string{"link": link, "timestamp": timestamp}),
		).Set(float64(rbytes))
		c.oBytes.With(
			metric.NewLabels(map[string]string{"link": link, "timestamp": timestamp}),
		).Set(float64(obytes))
		c.iErrors.With(
			metric.NewLabels(map[string]string{"link": link, "timestamp": timestamp}),
		).Set(float64(ierrors))
		c.oErrors.With(
			metric.NewLabels(map[string]string{"link": link, "timestamp": timestamp}),
		).Set(float64(oerrors))
		ipackets = ipackets
		opackets = opackets
		rbytes = rbytes
		obytes = obytes
		ierrors = ierrors
		oerrors = oerrors
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

	c.ResetVectors()
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

func (c *netCollector) ResetVectors() {
	c.iPackets.Reset()
	c.oPackets.Reset()
	c.rBytes.Reset()
	c.oBytes.Reset()
	c.iErrors.Reset()
	c.oErrors.Reset()
	c.class.Reset()
	c.mtu.Reset()
	c.state.Reset()
	c.bridge.Reset()
	c.over.Reset()
}

func parseDladmOutput(out string) error {
	//var err error
	outlines := strings.Split(out, "\n")
	for _, l := range outlines {
		values := strings.Split(l, ":")
		if values[0] == "" {
			continue
		}
		fmt.Print(values)
		fmt.Print("\n")
	}
	return nil
}
