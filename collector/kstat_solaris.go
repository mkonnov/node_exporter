package collector

import (
	"fmt"
	"math"

	//	"fmt"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	kstat "github.com/illumos/go-kstat"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"strings"
)

const kStatSnaptime = "snaptime"

type kstatStat struct {
	ID          string
	scaleFactor float64
	suffix      string
	help        string
}

type kstatName struct {
	ID    re
	label string
	stats []kstatStat
}

type kstatModule struct {
	ID    string
	names []kstatName
}

type kstatCollector struct {
	modules []kstatModule
	logger  log.Logger
}

func init() {
	registerCollector("kstat", defaultEnabled, NewKstatCollector)
}

func NewKstatCollector(logger log.Logger) (Collector, error) {
	var (
		c   kstatCollector
		cfg kstatConfig
	)

	err := cfg.init()
	if err != nil {
		return nil, err
	}

	for _, cfgModule := range cfg.KstatModules {
		module := kstatModule{ID: cfgModule.ID}
		for _, cfgName := range cfgModule.KstatNames {
			name := kstatName{ID: cfgName.ID, label: cfgName.LabelString}
			for _, cfgStat := range cfgName.KstatStats {
				stat := kstatStat{
					ID:          cfgStat.ID,
					scaleFactor: cfgStat.ScaleFactor,
					suffix:      cfgStat.Suffix,
					help:        cfgStat.Help,
				}
				name.stats = append(name.stats, stat)
			}

			//Snaptime is separate kind because of
			//different way to retrieve this metric
			stat := kstatStat{
				ID: kStatSnaptime,
			}
			name.stats = append(name.stats, stat)

			module.names = append(module.names, name)
		}
		c.modules = append(c.modules, module)
	}

	c.logger = logger

	return &c, nil
}

func (c *kstatCollector) throwError(module string, name string, stat string, inst int, err error) {
	level.Error(c.logger).Log(module+":"+strconv.Itoa(inst)+":"+name+":"+stat, err)
}

func (c *kstatCollector) Update(ch chan<- prometheus.Metric) error {
	var (
		tok         *kstat.Token
		ks          *kstat.KStat
		named       *kstat.Named
		vminfo      *kstat.Vminfo
		err         error
		metricValue float64
		vminfoDict  map[string]uint64
	)

	tok, err = kstat.Open()
	if err != nil {
		return err
	}

	defer tok.Close()

	for _, module := range c.modules {
		for _, name := range module.names {
			//Workaround for non-named kstats
			if module.ID == "unix" && name.ID.MatchString("vminfo") {
				ks, vminfo, err = tok.Vminfo()
				ks = ks
				if err != nil {
					c.throwError(module.ID, name.ID.String(), "", 0, err)
					break
				}

				vminfoDict = map[string]uint64{
					"freemem":    vminfo.Freemem,
					"swap_alloc": vminfo.Alloc,
					"swap_avail": vminfo.Avail,
					"swap_free":  vminfo.Free,
					"swap_resv":  vminfo.Resv,
					"updates":    vminfo.Updates,
				}

				for _, stat := range name.stats {
					desc := createTypedMetricDescription(
						prometheus.CounterValue,
						module.ID,
						"vminfo",
						formatMetricName(stat.ID, stat.suffix),
						name.label,
						stat.help,
					)
					ch <- desc.mustNewConstMetric(
						float64(vminfoDict[stat.ID])*stat.scaleFactor, "0")
				}
				continue
			}

			for inst := 0; ; inst++ {
				i := inst
				level.Debug(c.logger).Log(
					"msg", "looking up kstat",
					"module", module.ID,
					"instance", i,
					"name", name.ID.String(),
				)
				query := kstat.NewKStatQuery(kstat.MatchableString(module.ID), &i, name.ID)
				ksNames, err := tok.List(query)
				if err != nil {
					//Handle the instance number out-of-bound error
					break
				}
				for _, ksName := range ksNames {
					for _, stat := range name.stats {
						var desc typedDesc
						if strings.HasSuffix(stat.ID, kStatSnaptime) {
							metricValue = float64(ksName.Snaptime)
							desc = createSnapTimeMetricDescription(ksName.Module, stat.ID)
						} else {
							named, err = ksName.GetNamed(stat.ID)
							if err != nil {
								c.throwError(module.ID, ksName.Name, stat.ID, inst, err)
								continue
							}
							metricValue = float64(named.UintVal) * stat.scaleFactor
							desc = createTypedMetricDescription(
								prometheus.CounterValue,
								module.ID,
								ksName.Name,
								formatMetricName(stat.ID, stat.suffix),
								name.label,
								stat.help,
							)
						}

						//Round the value down to the number integer value
						//like 2.45 to 2.0. At the same time we have
						//to stick to float64 type.
						ch <- desc.mustNewConstMetric(
							math.Floor(metricValue),
							strconv.Itoa(inst),
						)
					}
				}
			}
		}
	}
	return nil
}

func createSnapTimeMetricDescription(module, name string) typedDesc {
	desc := prometheus.NewDesc(
		prometheus.BuildFQName(
			namespace,
			formatSubsystemName(module, name),
			kStatSnaptime,
		),
		formatSnapTimeMetricName(module, name),
		[]string{"inst"},
		nil,
	)
	return typedDesc{desc, prometheus.CounterValue}
}

func createTypedMetricDescription(
	valueType prometheus.ValueType,
	module,
	name,
	stat,
	label,
	help string,
) typedDesc {
	return typedDesc{
		desc:      createMetricDescription(module, name, stat, label, help),
		valueType: valueType,
	}
}

func createMetricDescription(module, name, stat, label, help string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(
			namespace,
			formatSubsystemName(module, name),
			stat,
		),
		help,
		[]string{label},
		nil,
	)
}

func formatSubsystemName(module, name string) string {
	return fmt.Sprintf(
		"kstat_%s_%s",
		hyphenToUnderscore(module),
		hyphenToUnderscore(name),
	)
}

func formatSnapTimeMetricName(module, name string) string {
	return fmt.Sprintf("%s::%s:%s", module, name, kStatSnaptime)
}

func formatMetricName(name, suffix string) string {
	return fmt.Sprintf("%s_%s", hyphenToUnderscore(name), suffix)
}

func hyphenToUnderscore(s string) string {
	return strings.ReplaceAll(s, "-", "_")
}
