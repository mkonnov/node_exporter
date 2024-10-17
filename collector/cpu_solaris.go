// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build exclude

package collector

import (
	"strconv"

	"github.com/go-kit/log"
	"github.com/illumos/go-kstat"
	"github.com/prometheus/client_golang/prometheus"
)

// #include <unistd.h>
import "C"

type cpuCollector struct {
	cpu_seconds    typedDesc
	cpu_ticks      typedDesc
	cpu_load_intr  typedDesc
	cpumigrate     typedDesc
	iowait         typedDesc
	nthreads       typedDesc
	syscall        typedDesc
	sysexec        typedDesc
	sysfork        typedDesc
	sysread        typedDesc
	sysvfork       typedDesc
	syswrite       typedDesc
	trap           typedDesc
	idlethread     typedDesc
	intrblk        typedDesc
	intrthread     typedDesc
	inv_swtch      typedDesc
	mutex_adenters typedDesc
	xcalls         typedDesc
	logger         log.Logger
}

func init() {
	registerCollector("cpu", defaultEnabled, NewCpuCollector)
}

func NewCpuCollector(logger log.Logger) (Collector, error) {
	return &cpuCollector{
		cpu_seconds: typedDesc{nodeCPUSecondsDesc,
			prometheus.CounterValue},

		cpu_ticks: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "ticks_total"),
				"Ticks the CPUs spent in each mode.",
				[]string{"cpu", "mode"}, nil,
			), prometheus.CounterValue},

		cpu_load_intr: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "load_intr_percents"),
				"Interrupt load factor, percents.",
				[]string{"cpu"}, nil,
			), prometheus.GaugeValue},

		cpumigrate: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "cpumigrate_total"),
				"CPU migrations by threads.",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		iowait: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "iowait_total"),
				"Procs waiting for block I/O.",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		nthreads: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "nthreads_total"),
				"thread_create()s.",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		syscall: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "syscall_total"),
				"system calls.",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		sysexec: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "sysexec_total"),
				"sysexec's.",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		sysfork: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "sysfork_total"),
				"forks.",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		sysread: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "sysread_total"),
				"read() + readv() system calls.",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		sysvfork: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "sysvfork_total"),
				"vforks.",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		syswrite: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "syswrite_total"),
				"write() + writev() system calls.",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		trap: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "trap_total"),
				"traps.",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		idlethread: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "idlethread_total"),
				"times idle thread scheduled.",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		intrblk: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "intrblk_total"),
				"ints blkd/prempted/rel'd (swtch).",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		intrthread: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "intrthread_total"),
				"interrupts as threads (below clock).",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		inv_swtch: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "inv_swtch_total"),
				"involuntary context switches.",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		mutex_adenters: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "mutex_adenters_total"),
				"failed mutex enters (adaptive)	.",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		xcalls: typedDesc{
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, cpuCollectorSubsystem, "xcalls_total"),
				"xcalls to other cpus.",
				[]string{"cpu"}, nil,
			), prometheus.CounterValue},

		logger: logger,
	}, nil
}

func (c *cpuCollector) Update(ch chan<- prometheus.Metric) error {

	var (
		kstatValue *kstat.Named
		err        error
	)

	ncpus := C.sysconf(C._SC_NPROCESSORS_ONLN)

	tok, err := kstat.Open()
	if err != nil {
		return err
	}

	defer tok.Close()

	for cpu := 0; cpu < int(ncpus); cpu++ {
		ksCPU, err := tok.Lookup("cpu", cpu, "sys")
		if err != nil {
			goto exit
		}

		for k, v := range map[string]string{
			"idle":   "cpu_nsec_idle",
			"kernel": "cpu_nsec_kernel",
			"user":   "cpu_nsec_user",
			"intr":   "cpu_nsec_intr",
			"dtrace": "cpu_nsec_dtrace",
		} {
			kstatValue, err = ksCPU.GetNamed(v)
			if err != nil {
				goto exit
			}
			ch <- c.cpu_seconds.mustNewConstMetric(
				float64(kstatValue.UintVal)/1e9, strconv.Itoa(cpu), k)
		}

		for k, v := range map[string]string{
			"idle":   "cpu_ticks_idle",
			"kernel": "cpu_ticks_kernel",
			"user":   "cpu_ticks_user",
			"intr":   "cpu_ticks_wait",
		} {
			kstatValue, err = ksCPU.GetNamed(v)
			if err != nil {
				goto exit
			}
			ch <- c.cpu_ticks.mustNewConstMetric(
				float64(kstatValue.UintVal), strconv.Itoa(cpu), k)
		}

		for k, inst := range map[string]typedDesc{
			"cpu_load_intr":  c.cpu_load_intr,
			"cpumigrate":     c.cpumigrate,
			"iowait":         c.iowait,
			"nthreads":       c.nthreads,
			"syscall":        c.syscall,
			"sysexec":        c.sysexec,
			"sysfork":        c.sysfork,
			"sysread":        c.sysread,
			"sysvfork":       c.sysvfork,
			"syswrite":       c.syswrite,
			"trap":           c.trap,
			"idlethread":     c.idlethread,
			"intrblk":        c.intrblk,
			"intrthread":     c.intrthread,
			"inv_swtch":      c.inv_swtch,
			"mutex_adenters": c.mutex_adenters,
			"xcalls":         c.xcalls,
		} {
			kstatValue, err = ksCPU.GetNamed(k)
			if err != nil {
				goto exit
			}
			ch <- inst.mustNewConstMetric(
				float64(kstatValue.UintVal), strconv.Itoa(cpu))
		}
	}
exit:
	if err != nil {
		return err
	}
	return nil
}
