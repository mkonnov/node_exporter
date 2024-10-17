// zpool collector
// this will :
//  - call zpool list
//  - gather ZPOOL metrics
//  - feed the collector

package collector

import (
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"os/exec"
	"strconv"
	"strings"
	"time"
	// Prometheus Go toolset
	"github.com/prometheus/client_golang/prometheus"
)

const (
	NSEC_TO_SEC_FACTOR = float64(1) / (1000 * 1000 * 1000)
)

func init() {
	registerCollector("zpool", defaultEnabled, NewGZZpoolListExporter)
}

// GZZpoolListCollector declares the data type within the prometheus metrics package.
type GZZpoolListCollector struct {
	gzZpoolListAlloc   *prometheus.GaugeVec
	gzZpoolListFrag    *prometheus.GaugeVec
	gzZpoolListFree    *prometheus.GaugeVec
	gzZpoolListSize    *prometheus.GaugeVec
	gzZpoolListFreeing *prometheus.GaugeVec
	gzZpoolListHealth  *prometheus.GaugeVec
	gzZpoolListLeaked  *prometheus.GaugeVec
	gzZpoolListGuid    *prometheus.GaugeVec

	gzZfsLogicalUsed *prometheus.GaugeVec

	gzZpoolListCapacityAlloc    *prometheus.GaugeVec
	gzZpoolListCapacityFree     *prometheus.GaugeVec
	gzZpoolListOperationsRead   *prometheus.GaugeVec
	gzZpoolListOperationsWrite  *prometheus.GaugeVec
	gzZpoolListBandwidthRead    *prometheus.GaugeVec
	gzZpoolListBandWidthWrite   *prometheus.GaugeVec
	gzZpoolListTotalWaitRead    *prometheus.GaugeVec
	gzZpoolListTotalWaitWrite   *prometheus.GaugeVec
	gzZpoolListDiskRead         *prometheus.GaugeVec
	gzZpoolListDiskWrite        *prometheus.GaugeVec
	gzZpoolListSyncWaitRead     *prometheus.GaugeVec
	gzZpoolListSyncWaitWrite    *prometheus.GaugeVec
	gzZpoolListAsyncWaitRead    *prometheus.GaugeVec
	gzZpoolListAsyncWaitWrite   *prometheus.GaugeVec
	gzZpoolListScrubWait        *prometheus.GaugeVec
	gzZpoolListTrimWait         *prometheus.GaugeVec
	gzZpoolListSyncqReadPend    *prometheus.GaugeVec
	gzZpoolListSyncqReadActiv   *prometheus.GaugeVec
	gzZpoolListSyncqWritePend   *prometheus.GaugeVec
	gzZpoolListSyncqWriteActiv  *prometheus.GaugeVec
	gzZpoolListAsyncqReadPend   *prometheus.GaugeVec
	gzZpoolListAsyncqReadActiv  *prometheus.GaugeVec
	gzZpoolListAsyncqWritePend  *prometheus.GaugeVec
	gzZpoolListAsyncqWriteActiv *prometheus.GaugeVec
	gzZpoolListScrubqReadPend   *prometheus.GaugeVec
	gzZpoolListScrubqReadActiv  *prometheus.GaugeVec
	gzZpoolListTrimqWritePend   *prometheus.GaugeVec
	gzZpoolListTrimqWriteActiv  *prometheus.GaugeVec

	gzZpoolIostatSyncRead   *prometheus.GaugeVec
	gzZpoolIostatSyncWrite  *prometheus.GaugeVec
	gzZpoolIostatAsyncRead  *prometheus.GaugeVec
	gzZpoolIostatAsyncWrite *prometheus.GaugeVec
	gzZpoolIostatReadTotal  *prometheus.GaugeVec
	gzZpoolIostatWriteTotal *prometheus.GaugeVec

	logger log.Logger
}

// NewGZZpoolListExporter returns a newly allocated exporter GZZpoolListCollector.
// It exposes the zpool list command result.
func NewGZZpoolListExporter(logger log.Logger) (Collector, error) {

	return &GZZpoolListCollector{

		gzZpoolListAlloc: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "zpool_alloc_mbytes",
			Help: "zpool allocated, megabytes.",
		}, []string{"zpool"}),

		gzZpoolListFrag: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "zpool_frag_percents",
			Help: "zpool fragmentation, percents.",
		}, []string{"zpool"}),

		gzZpoolListFree: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "zpool_free_mbytes",
			Help: "zpool free, megabytes.",
		}, []string{"zpool"}),

		gzZpoolListSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "zpool_size_mbytes",
			Help: "zpool size, megabytes.",
		}, []string{"zpool"}),

		gzZpoolListFreeing: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "zpool_freeing_mbytes",
			Help: "zpool freeing, megabytes.",
		}, []string{"zpool"}),

		gzZpoolListHealth: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "zpool_health",
			Help: "zpool health status (0: OFFLINE, 1: ONLINE)",
		}, []string{"zpool"}),

		gzZpoolListLeaked: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "zpool_leaked_mbytes",
			Help: "zpool leaked mbytes.",
		}, []string{"zpool"}),

		gzZpoolListGuid: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "zpool_guid",
			Help: "zpool guid.",
		}, []string{"zpool", "guid"}),

		gzZfsLogicalUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "zpool_zfs_logicalused_mbytes",
			Help: "zfs logicalused.",
		}, []string{"zpool"}),

		gzZpoolListCapacityAlloc: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_capacity_alloc_bytes"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListCapacityFree: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_capacity_free_bytes"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListOperationsRead: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_operations_read_number"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListOperationsWrite: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_operations_write_number"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		//gzZpoolListBandwidthRead: prometheus.NewGaugeVec(prometheus.GaugeOpts{
		//	Name: prometheus.BuildFQName(namespace, "zpool", "iostat_bandwidth_read_bytes"),
		//	Help: "zpool iostat .",
		//}, []string{"pool", "vdev"}),
		//
		//gzZpoolListBandWidthWrite: prometheus.NewGaugeVec(prometheus.GaugeOpts{
		//	Name: prometheus.BuildFQName(namespace, "zpool", "iostat_bandwidth_write_bytes"),
		//	Help: "zpool iostat .",
		//}, []string{"pool", "vdev"}),

		gzZpoolListTotalWaitRead: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_total_wait_read_seconds"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListTotalWaitWrite: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_total_wait_write_seconds"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListDiskRead: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_disk_wait_read_seconds"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListDiskWrite: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_disk_wait_write_seconds"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListSyncWaitRead: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_sync_wait_read_seconds"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListSyncWaitWrite: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_sync_wait_write_seconds"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListAsyncWaitRead: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_async_wait_read_seconds"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListAsyncWaitWrite: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_async_wait_write_seconds"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListScrubWait: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_scrub_wait_seconds"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListTrimWait: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_trim_wait_seconds"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListSyncqReadPend: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_syncq_read_pend_number"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListSyncqReadActiv: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_syncq_read_activ_number"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListSyncqWritePend: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_syncq_write_pend_number"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListSyncqWriteActiv: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_syncq_write_activ_number"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListAsyncqReadPend: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_asyncq_read_pend_number"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListAsyncqReadActiv: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_asyncq_read_activ_number"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListAsyncqWritePend: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_asyncq_write_pend_number"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListAsyncqWriteActiv: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_asyncq_write_activ_number"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListScrubqReadPend: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_scrubq_read_pend_number"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListScrubqReadActiv: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_scrubq_read_activ_number"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListTrimqWritePend: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_trimq_write_pend_number"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolListTrimqWriteActiv: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_trimq_write_activ_number"),
			Help: "zpool iostat .",
		}, []string{"pool", "vdev"}),

		gzZpoolIostatSyncRead: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_sync_read_operations_total"),
			Help: "zpool iostat sync read operations (ind + agg)",
		}, []string{"req_size", "pool", "timestamp"}),

		gzZpoolIostatSyncWrite: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_sync_write_operations_total"),
			Help: "zpool iostat sync write operations (ind + agg)",
		}, []string{"req_size", "pool", "timestamp"}),

		gzZpoolIostatAsyncRead: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_async_read_operations_total"),
			Help: "zpool iostat async read operations (ind + agg)",
		}, []string{"req_size", "pool", "timestamp"}),

		gzZpoolIostatAsyncWrite: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_async_write_operations_total"),
			Help: "zpool iostat async write operations (ind + agg)",
		}, []string{"req_size", "pool", "timestamp"}),

		gzZpoolIostatReadTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_read_bytes_total"),
			Help: "Sum of async + sync read operations. Each operation count (ind + agg) is multiplied by its related operation size.",
		}, []string{"pool", "timestamp"}),

		gzZpoolIostatWriteTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "zpool", "iostat_write_bytes_total"),
			Help: "Sum of async + sync write operations. Each operation count (ind + agg) is multiplied by its related operation size.",
		}, []string{"pool", "timestamp"}),

		logger: logger,
	}, nil
}

// Describe describes all the metrics.
func (e *GZZpoolListCollector) Describe(ch chan<- *prometheus.Desc) {
	e.gzZpoolListAlloc.Describe(ch)
	e.gzZpoolListFrag.Describe(ch)
	e.gzZpoolListFree.Describe(ch)
	e.gzZpoolListSize.Describe(ch)
	e.gzZpoolListFreeing.Describe(ch)
	e.gzZpoolListHealth.Describe(ch)
	e.gzZpoolListLeaked.Describe(ch)
	e.gzZpoolListGuid.Describe(ch)
	e.gzZfsLogicalUsed.Describe(ch)

	e.gzZpoolListCapacityAlloc.Describe(ch)
	e.gzZpoolListCapacityFree.Describe(ch)
	e.gzZpoolListOperationsRead.Describe(ch)
	e.gzZpoolListOperationsWrite.Describe(ch)
	//e.gzZpoolListBandwidthRead.Describe(ch)
	//e.gzZpoolListBandWidthWrite.Describe(ch)
	e.gzZpoolListTotalWaitRead.Describe(ch)
	e.gzZpoolListTotalWaitWrite.Describe(ch)
	e.gzZpoolListDiskRead.Describe(ch)
	e.gzZpoolListDiskWrite.Describe(ch)
	e.gzZpoolListSyncWaitRead.Describe(ch)
	e.gzZpoolListSyncWaitWrite.Describe(ch)
	e.gzZpoolListAsyncWaitRead.Describe(ch)
	e.gzZpoolListAsyncWaitWrite.Describe(ch)
	e.gzZpoolListScrubWait.Describe(ch)
	e.gzZpoolListTrimWait.Describe(ch)
	e.gzZpoolListSyncqReadPend.Describe(ch)
	e.gzZpoolListSyncqReadActiv.Describe(ch)
	e.gzZpoolListSyncqWritePend.Describe(ch)
	e.gzZpoolListSyncqWriteActiv.Describe(ch)
	e.gzZpoolListAsyncqReadPend.Describe(ch)
	e.gzZpoolListAsyncqReadActiv.Describe(ch)
	e.gzZpoolListAsyncqWritePend.Describe(ch)
	e.gzZpoolListAsyncqWriteActiv.Describe(ch)
	e.gzZpoolListScrubqReadPend.Describe(ch)
	e.gzZpoolListScrubqReadActiv.Describe(ch)
	e.gzZpoolListTrimqWritePend.Describe(ch)
	e.gzZpoolListTrimqWriteActiv.Describe(ch)

	e.gzZpoolIostatSyncRead.Describe(ch)
	e.gzZpoolIostatSyncWrite.Describe(ch)
	e.gzZpoolIostatAsyncRead.Describe(ch)
	e.gzZpoolIostatAsyncWrite.Describe(ch)

	e.gzZpoolIostatReadTotal.Describe(ch)
	e.gzZpoolIostatWriteTotal.Describe(ch)
}

// Collect fetches the stats.
func (e *GZZpoolListCollector) Update(ch chan<- prometheus.Metric) error {

	e.zpoolGetProps()
	e.zfsGetProps()
	e.zpoolIostatLatenciesQueues()
	e.zpoolIostatRequestSizes()

	e.gzZpoolListAlloc.Collect(ch)
	e.gzZpoolListFrag.Collect(ch)
	e.gzZpoolListFree.Collect(ch)
	e.gzZpoolListSize.Collect(ch)
	e.gzZpoolListFreeing.Collect(ch)
	e.gzZpoolListHealth.Collect(ch)
	e.gzZpoolListLeaked.Collect(ch)
	e.gzZpoolListGuid.Collect(ch)
	e.gzZfsLogicalUsed.Collect(ch)

	e.gzZpoolListCapacityAlloc.Collect(ch)
	e.gzZpoolListCapacityFree.Collect(ch)
	e.gzZpoolListOperationsRead.Collect(ch)
	e.gzZpoolListOperationsWrite.Collect(ch)
	//e.gzZpoolListBandwidthRead.Collect(ch)
	//e.gzZpoolListBandWidthWrite.Collect(ch)
	e.gzZpoolListTotalWaitRead.Collect(ch)
	e.gzZpoolListTotalWaitWrite.Collect(ch)
	e.gzZpoolListDiskRead.Collect(ch)
	e.gzZpoolListDiskWrite.Collect(ch)
	e.gzZpoolListSyncWaitRead.Collect(ch)
	e.gzZpoolListSyncWaitWrite.Collect(ch)
	e.gzZpoolListAsyncWaitRead.Collect(ch)
	e.gzZpoolListAsyncWaitWrite.Collect(ch)
	e.gzZpoolListScrubWait.Collect(ch)
	e.gzZpoolListTrimWait.Collect(ch)
	e.gzZpoolListSyncqReadPend.Collect(ch)
	e.gzZpoolListSyncqReadActiv.Collect(ch)
	e.gzZpoolListSyncqWritePend.Collect(ch)
	e.gzZpoolListSyncqWriteActiv.Collect(ch)
	e.gzZpoolListAsyncqReadPend.Collect(ch)
	e.gzZpoolListAsyncqReadActiv.Collect(ch)
	e.gzZpoolListAsyncqWritePend.Collect(ch)
	e.gzZpoolListAsyncqWriteActiv.Collect(ch)
	e.gzZpoolListScrubqReadPend.Collect(ch)
	e.gzZpoolListScrubqReadActiv.Collect(ch)
	e.gzZpoolListTrimqWritePend.Collect(ch)
	e.gzZpoolListTrimqWriteActiv.Collect(ch)

	e.gzZpoolIostatSyncRead.Collect(ch)
	e.gzZpoolIostatSyncWrite.Collect(ch)
	e.gzZpoolIostatAsyncRead.Collect(ch)
	e.gzZpoolIostatAsyncWrite.Collect(ch)

	e.gzZpoolIostatReadTotal.Collect(ch)
	e.gzZpoolIostatWriteTotal.Collect(ch)

	return nil
}

func (e *GZZpoolListCollector) zpoolGetProps() error {
	out, eerr := exec.Command("zpool", "get", "-Hp",
		"size,free,allocated,fragmentation,freeing,health,allocated,leaked,guid").Output()
	if eerr != nil {
		level.Error(e.logger).Log("error on executing zpool get: %v", eerr)
	} else {
		perr := e.parseZpoolGetPropsOutput(string(out))
		if perr != nil {
			level.Error(e.logger).Log("error on parsing zpool get output: %v", perr)
		}
	}
	return nil
}

func (e *GZZpoolListCollector) zpoolIostatLatenciesQueues() error {
	out, eerr := exec.Command("zpool", "iostat", "-plqv").Output()
	if eerr != nil {
		level.Error(e.logger).Log("error on executing zpool iostat: %v", eerr)
	} else {
		perr := e.parseZpoolIostatLatenciesQueuesOutput(string(out))
		if perr != nil {
			level.Error(e.logger).Log("error on parsing zpool iostat output: %v", perr)
		}
	}
	return nil
}

func (e *GZZpoolListCollector) zpoolIostatRequestSizes() error {
	timestamp := time.Now().UnixNano() / 1e6
	out, eerr := exec.Command("zpool", "iostat", "-pr").Output()
	if eerr != nil {
		level.Error(e.logger).Log("error on executing zpool iostat: %v", eerr)
	} else {
		perr := e.parseZpoolIostatRequestSizesOutput(string(out), timestamp)
		if perr != nil {
			level.Error(e.logger).Log("error on parsing zpool iostat output: %v", perr)
		}
	}
	return nil
}

// Yes, zfs get. Though we already have a dedicated collector for zfs,
// but here we need to retrieve only some pool-related statistics
func (e *GZZpoolListCollector) zfsGetProps() error {
	out, eerr := exec.Command("zfs", "get", "-Hp", "logicalused").Output()
	if eerr != nil {
		level.Error(e.logger).Log("error on executing zfs: %v", eerr)
	} else {
		perr := e.parseZfsGetOutput(string(out))
		if perr != nil {
			level.Error(e.logger).Log("error on parsing zpool: %v", perr)
		}
	}
	return nil
}

// rpool   size    33822867456     -
func (e *GZZpoolListCollector) parseZpoolGetPropsOutput(out string) error {
	outlines := strings.Split(out, "\n")
	l := len(outlines)

	for _, line := range outlines[0 : l-1] {
		parsed_line := strings.Fields(line)
		pool_name := parsed_line[0]
		val := parsed_line[2]

		//Filter out the lines with value presented as "-"
		if val == "-" {
			continue
		}

		switch parsed_line[1] {
		case "size":
			pval, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return err
			}
			e.gzZpoolListSize.With(prometheus.Labels{"zpool": pool_name}).Set(pval / 1024 / 1024)
		case "free":
			pval, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return err
			}
			e.gzZpoolListFree.With(prometheus.Labels{"zpool": pool_name}).Set(pval / 1024 / 1024)
		case "allocated":
			pval, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return err
			}
			e.gzZpoolListAlloc.With(prometheus.Labels{"zpool": pool_name}).Set(pval / 1024 / 1024)
		case "fragmentation":
			pval, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return err
			}
			e.gzZpoolListFrag.With(prometheus.Labels{"zpool": pool_name}).Set(pval)
		case "freeing":
			pval, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return err
			}
			e.gzZpoolListFreeing.With(prometheus.Labels{"zpool": pool_name}).Set(pval)
		case "health":
			if (strings.Contains(val, "ONLINE")) == true {
				e.gzZpoolListHealth.With(prometheus.Labels{"zpool": pool_name}).Set(1)
			} else {
				e.gzZpoolListHealth.With(prometheus.Labels{"zpool": pool_name}).Set(0)
			}
		case "leaked":
			pval, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return err
			}
			e.gzZpoolListFrag.With(prometheus.Labels{"zpool": pool_name}).Set(pval / 1024 / 1024)

		case "guid":
			e.gzZpoolListGuid.With(prometheus.Labels{"zpool": pool_name, "guid": val}).Set(1)
		}
	}
	return nil
}

func (e *GZZpoolListCollector) parseZfsGetOutput(out string) error {
	outlines := strings.Split(out, "\n")
	l := len(outlines)
	for _, line := range outlines[0 : l-1] {
		parsed_line := strings.Fields(line)
		name := parsed_line[0]
		if strings.Contains(name, "/") {
			continue
		}
		if parsed_line[2] == "-" {
			continue
		}
		pval, err := strconv.ParseFloat(parsed_line[2], 64)
		if err != nil {
			level.Error(e.logger).Log("error on parsing zfs output: %v", err)
			continue
		}
		e.gzZfsLogicalUsed.With(prometheus.Labels{"zpool": name}).Set(float64(pval / 1024 / 1024))
	}
	return nil
}

func (e *GZZpoolListCollector) helperZpoolWithSet(pool string, vdev string,
	vec *prometheus.GaugeVec, val string, scale float64) error {
	var pval float64
	var err error
	if val == "-" {
		pval = -1
		err = nil
	} else {
		pval, err = strconv.ParseFloat(val, 64)
		if err != nil {
			level.Error(e.logger).Log("error on parsing zpool iostat: %v", err)
			return err
		}
		pval = pval * scale
	}
	vec.With(prometheus.Labels{"pool": pool, "vdev": vdev}).Set(pval)
	return nil
}

func parsePrefix(prefix string) (int, error) {
	switch prefix {
	case "512":
		return 512, nil
	case "1K":
		return (1 * 1024), nil
	case "2K":
		return (2 * 1024), nil
	case "4K":
		return (4 * 1024), nil
	case "8K":
		return (8 * 1024), nil
	case "16K":
		return (16 * 1024), nil
	case "32K":
		return (32 * 1024), nil
	case "64K":
		return (64 * 1024), nil
	case "128K":
		return (128 * 1024), nil
	case "256K":
		return (256 * 1024), nil
	case "512K":
		return (512 * 1024), nil
	case "1M":
		return (1 * 1024 * 1024), nil
	case "2M":
		return (2 * 1024 * 1024), nil
	case "4M":
		return (4 * 1024 * 1024), nil
	case "8M":
		return (8 * 1024 * 1024), nil
	case "16M":
		return (16 * 1024 * 1024), nil
	}
	return 0, error(nil)
}

func (e *GZZpoolListCollector) parseZpoolIostatRequestSizesOutput(out string, timestamp int64) error {
	//var pool, vdev string
	//var err error
	tables := strings.Split(out[1:len(out)], "\n\n")

	e.gzZpoolIostatSyncRead.Reset()
	e.gzZpoolIostatSyncWrite.Reset()
	e.gzZpoolIostatAsyncRead.Reset()
	e.gzZpoolIostatAsyncWrite.Reset()
	e.gzZpoolIostatReadTotal.Reset()
	e.gzZpoolIostatWriteTotal.Reset()

	for _, table := range tables {
		bytes_read_total := 0
		bytes_write_total := 0
		outlines := strings.Split(table, "\n")
		l := len(outlines)
		if l == 0 {
			return error(nil)
		}
		pool := strings.Fields(outlines[0])[0]
		for _, line := range outlines[2:l] {
			if len(line) == 0 {
				continue
			}
			if strings.HasPrefix(line, "-") {
				continue
			}

			parsed_line := strings.Fields(line)
			req_size, err := parsePrefix(parsed_line[0])

			ind, err := strconv.Atoi(parsed_line[1])
			if err != nil {
				return err
			}
			agg, err := strconv.Atoi(parsed_line[2])
			if err != nil {
				return err
			}
			ctr_sync_read := ind + agg

			ind, err = strconv.Atoi(parsed_line[3])
			if err != nil {
				return err
			}
			agg, err = strconv.Atoi(parsed_line[4])
			if err != nil {
				return err
			}
			ctr_sync_write := ind + agg

			ind, err = strconv.Atoi(parsed_line[5])
			if err != nil {
				return err
			}
			agg, err = strconv.Atoi(parsed_line[6])
			if err != nil {
				return err
			}
			ctr_async_read := ind + agg

			ind, err = strconv.Atoi(parsed_line[7])
			if err != nil {
				return err
			}
			agg, err = strconv.Atoi(parsed_line[8])
			if err != nil {
				return err
			}
			ctr_async_write := (ind + agg)

			bytes_read_total += (ctr_sync_read*req_size + ctr_async_read*req_size)
			bytes_write_total += (ctr_sync_write*req_size + ctr_async_write*req_size)

			e.gzZpoolIostatSyncRead.With(
				prometheus.Labels{
					"pool":      pool,
					"req_size":  parsed_line[0],
					"timestamp": strconv.FormatInt(timestamp, 10)}).Set(float64(ctr_sync_read))
			e.gzZpoolIostatSyncWrite.With(
				prometheus.Labels{
					"pool":      pool,
					"req_size":  parsed_line[0],
					"timestamp": strconv.FormatInt(timestamp, 10)}).Set(float64(ctr_sync_write))
			e.gzZpoolIostatAsyncRead.With(
				prometheus.Labels{
					"pool":      pool,
					"req_size":  parsed_line[0],
					"timestamp": strconv.FormatInt(timestamp, 10)}).Set(float64(ctr_async_read))
			e.gzZpoolIostatAsyncWrite.With(
				prometheus.Labels{
					"pool":      pool,
					"req_size":  parsed_line[0],
					"timestamp": strconv.FormatInt(timestamp, 10)}).Set(float64(ctr_async_write))
			if err != nil {
				return err
			}
		}
		e.gzZpoolIostatReadTotal.With(prometheus.Labels{
			"pool":      pool,
			"timestamp": strconv.FormatInt(timestamp, 10)}).Set(float64(bytes_read_total))
		e.gzZpoolIostatWriteTotal.With(prometheus.Labels{
			"pool":      pool,
			"timestamp": strconv.FormatInt(timestamp, 10)}).Set(float64(bytes_write_total))

	}
	return nil
}

func (e *GZZpoolListCollector) parseZpoolIostatLatenciesQueuesOutput(out string) error {
	var pool, vdev string
	var err error
	outlines := strings.Split(out, "\n")
	l := len(outlines)
	for _, line := range outlines[2:l] {
		vdev = "-"
		if strings.HasPrefix(line, "-") {
			continue
		}
		parsed_line := strings.Fields(line)
		if len(parsed_line) == 0 {
			continue
		}

		//Determine if its pool or vdev info string
		if !strings.HasPrefix(line, " ") {
			pool = parsed_line[0]
		} else {
			vdev = parsed_line[0]
		}
		/*
			0: name
			1: capacity alloc 	bytes
			2: capacity free 	bytes
			3: operations read 	num
			4: operations write 	num
			5: bandwidth read	bytes
			6: bandwidth write	bytes
			7: total_wait read	nanosec
			8: total_wait write	nanosec
			9: disk_wait read	nanosec
			10: disk_wait write	nanosec
			11: sync_wait read	nanosec
			12: sync_wait write	nanosec
			13: async_wait read	nanosec
			14: async_wait write	nanosec
			15: scrub_wait wait	nanosec
			16: trim wait		nanosec
			17: syncq_read pend	num
			18: syncq_read activ	num
			19: syncq_write pend	num
			20: syncq_write activ	num
			21: asyncq_read pend	num
			22: asyncq_read activ	num
			23: asyncq_write pend	num
			24: asyncq_write activ	num
			25: scrubq_read pend 	num
			26: scrubq_read activ	num
			27: trimq_write pend	num
			28: trimq_write activ 	num
		*/

		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListCapacityAlloc, parsed_line[1], 1)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListCapacityFree, parsed_line[2], 1)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListOperationsRead, parsed_line[3], 1)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListOperationsWrite, parsed_line[4], 1)
		if err != nil {
			return err
		}
		//err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListBandwidthRead, parsed_line[5], 1)
		//if err != nil { return err }
		//err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListBandWidthWrite, parsed_line[6], 1)
		//if err != nil { return err }
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListTotalWaitRead, parsed_line[7], NSEC_TO_SEC_FACTOR)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListTotalWaitWrite, parsed_line[8], NSEC_TO_SEC_FACTOR)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListDiskRead, parsed_line[9], NSEC_TO_SEC_FACTOR)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListDiskWrite, parsed_line[10], NSEC_TO_SEC_FACTOR)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListSyncWaitRead, parsed_line[11], NSEC_TO_SEC_FACTOR)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListSyncWaitWrite, parsed_line[12], NSEC_TO_SEC_FACTOR)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListAsyncWaitRead, parsed_line[13], NSEC_TO_SEC_FACTOR)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListAsyncWaitWrite, parsed_line[14], NSEC_TO_SEC_FACTOR)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListScrubWait, parsed_line[15], NSEC_TO_SEC_FACTOR)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListTrimWait, parsed_line[16], NSEC_TO_SEC_FACTOR)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListSyncqReadPend, parsed_line[17], 1)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListSyncqReadActiv, parsed_line[18], 1)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListSyncqWritePend, parsed_line[19], 1)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListSyncqWriteActiv, parsed_line[20], 1)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListAsyncqReadPend, parsed_line[21], 1)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListAsyncqReadActiv, parsed_line[22], 1)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListAsyncqWritePend, parsed_line[23], 1)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListAsyncqWriteActiv, parsed_line[24], 1)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListScrubqReadPend, parsed_line[25], 1)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListScrubqReadActiv, parsed_line[26], 1)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListTrimqWritePend, parsed_line[27], 1)
		if err != nil {
			return err
		}
		err = e.helperZpoolWithSet(pool, vdev, e.gzZpoolListTrimqWriteActiv, parsed_line[28], 1)
		if err != nil {
			return err
		}
	}
	return nil
}
