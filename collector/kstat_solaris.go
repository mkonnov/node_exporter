package collector

import (
	//	"fmt"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/illumos/go-kstat"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"strings"
)

type kstatStat struct {
	ID          string
	scaleFactor float64
	desc        typedDesc
}

type kstatName struct {
	ID    string
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
		module := kstatModule{}
		module.ID = cfgModule.ID
		for _, cfgName := range cfgModule.KstatNames {
			name := kstatName{}
			name.ID = cfgName.ID
			for _, cfgStat := range cfgName.KstatStats {
				stat := kstatStat{}
				stat.ID = cfgStat.ID
				desc := prometheus.NewDesc(
					prometheus.BuildFQName(
						namespace,
						"kstat_"+strings.ReplaceAll(cfgModule.ID, "-", "_")+"_"+
							strings.ReplaceAll(cfgName.ID, "-", "_"),
						strings.ReplaceAll(cfgStat.ID, "-", "_")+"_"+cfgStat.Suffix),
					cfgStat.Help, []string{cfgName.LabelString}, nil)
				stat.desc = typedDesc{desc, prometheus.CounterValue}
				stat.scaleFactor = float64(cfgStat.ScaleFactor)
				name.stats = append(name.stats, stat)
			}

			//Snaptime is separate kind because of
			//different way to retrieve this metric
			stat := kstatStat{}
			stat.ID = "snaptime"
			desc := prometheus.NewDesc(
				prometheus.BuildFQName(
					namespace,
					"kstat_"+cfgModule.ID+"_"+cfgName.ID,
					"snaptime"),
				cfgModule.ID+"::"+cfgName.ID+":"+"snaptime",
				[]string{"inst"}, nil)
			stat.desc = typedDesc{desc, prometheus.CounterValue}
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
			if module.ID == "unix" && name.ID == "vminfo" {
				ks, vminfo, err = tok.Vminfo()
				ks = ks
				if err != nil {
					c.throwError(module.ID, name.ID, "", 0, err)
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
					ch <- stat.desc.mustNewConstMetric(
						float64(vminfoDict[stat.ID])*stat.scaleFactor, "0")
				}
				continue
			}
			inst := 0
			for {
				ksName, err := tok.Lookup(module.ID, inst, name.ID)
				if err != nil {
					//Handle the instance number out-of-bound error
					break
				}
				for _, stat := range name.stats {
					if strings.HasSuffix(stat.ID, "snaptime") {
						metricValue = float64(ksName.Snaptime)
					} else {
						named, err = ksName.GetNamed(stat.ID)
						if err != nil {
							c.throwError(module.ID, name.ID, stat.ID, inst, err)
							continue
						}
						metricValue = float64(named.UintVal) * stat.scaleFactor
					}

					//Round the value down to the number integer value
					//like 2.45 to 2.0. At the same time we have
					//to stick to float64 type.
					ch <- stat.desc.mustNewConstMetric(float64(int(metricValue)),
						strconv.Itoa(inst))
				}
				inst++
			}
		}
	}
	return nil
}
