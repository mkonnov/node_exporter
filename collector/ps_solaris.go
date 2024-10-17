//go:build !nops

package collector

import (
	"fmt"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os/exec"
	"slices"
	"strconv"
	"strings"
)

// #include <unistd.h>
import "C"

const PROCESS_MIN_NUM = 10

type PsCollector struct {
	psCpu         *prometheus.GaugeVec
	psMem         *prometheus.GaugeVec
	processNumCpu int
	processNumMem int
	logger        log.Logger
}

type psConfig struct {
	NumberCpu int `yaml:"number_cpu"`
	NumberMem int `yaml:"number_mem"`
}

type psLineDesc struct {
	pcpu   float64
	pmem   float64
	pid    uint
	zoneid uint
	comm   string
	rss    float64
	args   string
}

func init() {
	registerCollector("ps", defaultEnabled, NewPsCollector)
}

func NewPsCollector(logger log.Logger) (Collector, error) {
	var cfgFile psConfig
	//processNumCfgCpu := 10
	yamlFile, err := ioutil.ReadFile(psCfgFilePath())
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yamlFile, &cfgFile)
	if err != nil {
		return nil, err
	}

	processNumCfgMem := cfgFile.NumberMem
	processNumCfgCpu := cfgFile.NumberCpu

	ncpus := C.sysconf(C._SC_NPROCESSORS_ONLN)
	processMinNum := PROCESS_MIN_NUM

	if PROCESS_MIN_NUM < ncpus {
		processMinNum = int(ncpus)
		level.Warn(logger).Log("msg", "Minimum number of processes is less than number of CPUs",
			"processMinNum", processMinNum)
	}

	if processNumCfgCpu < processMinNum {
		level.Warn(logger).Log("msg", "Configured number of CPU processes is less than minumum required")
		processNumCfgCpu = processMinNum
	}

	if processNumCfgMem < processMinNum {
		level.Warn(logger).Log("msg", "Configured number of memory processes is less than minumum required")
		processNumCfgMem = processMinNum
	}

	return &PsCollector{
		psCpu: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "node_ps_top_cpu_percents",
			Help: "Process of top CPU consumption processes.",
		}, []string{"index", "pid", "zoneid", "comm", "args"}),
		psMem: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "node_ps_top_mem_kilobytes",
			Help: "Process of top memory consumption processes.",
		}, []string{"index", "pid", "zoneid", "comm", "args"}),
		processNumCpu: processNumCfgCpu,
		processNumMem: processNumCfgMem,
		logger:        logger,
	}, nil
}

func (c *PsCollector) Update(ch chan<- prometheus.Metric) error {
	c.getPsOut()

	c.psCpu.Collect(ch)
	c.psMem.Collect(ch)

	return nil
}

func (c *PsCollector) Describe(ch chan<- *prometheus.Desc) {
	c.psCpu.Describe(ch)
	c.psMem.Describe(ch)
}

func parsePsOutput(psOut string) ([]psLineDesc, error) {
	var err error
	var out []psLineDesc
	var args string

	psOutLines := strings.Split(psOut, "\n")

	for _, line := range psOutLines {
		//Filter out the header of ps output
		if strings.HasPrefix(line, "%") {
			continue
		}
		parsed_line := strings.Fields(line)
		if len(parsed_line) == 0 {
			continue
		}

		pcpu, err := strconv.ParseFloat(parsed_line[0], 64)
		if err != nil {
			goto exit
		}

		pmem, err := strconv.ParseFloat(parsed_line[0], 64)
		if err != nil {
			goto exit
		}

		pid, err := strconv.ParseUint(parsed_line[2], 10, 32)
		if err != nil {
			goto exit
		}

		zoneid, err := strconv.ParseUint(parsed_line[3], 10, 32)
		if err != nil {
			goto exit
		}

		comm := parsed_line[4]

		rss, err := strconv.ParseFloat(parsed_line[5], 64)
		if err != nil {
			goto exit
		}

		args = ""
		for i := range parsed_line[6:len(parsed_line)] {
			args += parsed_line[6+i] + " "
		}

		out = append(out, psLineDesc{
			pcpu:   pcpu,
			pmem:   pmem,
			pid:    uint(pid),
			zoneid: uint(zoneid),
			comm:   comm,
			rss:    rss,
			args:   args,
		})
	}
exit:
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (c *PsCollector) getPsOut() error {
	out, eerr := exec.Command("ps", "-eo", "pcpu,pmem,pid,zoneid,comm,rss,args").Output()
	if eerr != nil {
		level.Error(c.logger).Log("error on executing ps: %v", eerr)
	} else {
		psOutputParsed, perr := parsePsOutput(string(out))
		if perr != nil {
			level.Error(c.logger).Log("error on parsing ps out: %v", perr)
		}

		//the selectino must not exceed the number of processes in the output
		psLenCpu := min(len(psOutputParsed), c.processNumCpu)
		psLenMem := min(len(psOutputParsed), c.processNumMem)

		psCpuOutput := make([]psLineDesc, len(psOutputParsed))
		psMemOutput := make([]psLineDesc, len(psOutputParsed))

		copy(psCpuOutput, psOutputParsed)
		copy(psMemOutput, psOutputParsed)

		slices.SortFunc(psCpuOutput,
			func(a, b psLineDesc) int { return int(a.pcpu*100) - int(b.pcpu*100) })
		slices.SortFunc(psMemOutput,
			func(a, b psLineDesc) int { return int(a.rss) - int(b.rss) })

		//We must only leave only the last psLen items
		psCpuOutput = psCpuOutput[len(psCpuOutput)-psLenCpu:]
		psMemOutput = psMemOutput[len(psOutputParsed)-psLenMem:]

		slices.Reverse(psCpuOutput)

		c.psCpu.Reset()
		c.psMem.Reset()

		for i, l := range psCpuOutput {
			c.psCpu.With(prometheus.Labels{
				"index":  fmt.Sprintf("%d", i),
				"pid":    fmt.Sprintf("%d", l.pid),
				"zoneid": fmt.Sprintf("%d", l.zoneid),
				"comm":   l.comm,
				"args":   l.args,
			}).Set(l.pcpu)
		}

		for i, l := range psMemOutput {
			c.psMem.With(prometheus.Labels{
				"index":  fmt.Sprintf("%d", i),
				"pid":    fmt.Sprintf("%d", l.pid),
				"zoneid": fmt.Sprintf("%d", l.zoneid),
				"comm":   l.comm,
				"args":   l.args,
			}).Set(l.rss)
		}
	}
	return nil
}
