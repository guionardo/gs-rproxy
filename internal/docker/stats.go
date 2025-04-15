package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/client"
)

type (
	SummaryStats struct {
		UsedMemory      int64   // 		used_memory = memory_stats.usage - memory_stats.stats.cache
		AvailableMemory int64   //available_memory = memory_stats.limit
		MemoryUsage     float64 // Memory usage % = (used_memory / available_memory) * 100.0
		CPUDelta        int64   //cpu_delta = cpu_stats.cpu_usage.total_usage - precpu_stats.cpu_usage.total_usage
		SystemCPUDelta  int64   //system_cpu_delta = cpu_stats.system_cpu_usage - precpu_stats.system_cpu_usage
		NumberCPUs      int     //number_cpus = length(cpu_stats.cpu_usage.percpu_usage) or cpu_stats.online_cpus
		CPUUsage        float64 //CPU usage % = (cpu_delta / system_cpu_delta) * number_cpus * 100.0
		NetInput        int64
		NetOutput       int64
	}
	ContainerStats struct {
		Read     time.Time     `json:"read"`
		PreRead  time.Time     `json:"preread"`
		CPU      CPUStats      `json:"cpu_stats"`
		PreCPU   CPUStats      `json:""precpu_stats"`
		Mem      MemStats      `json:"memory_stats"`
		Networks NetworksStats `json:"networks"`
	}
	CPUStats struct {
		Usage       CPUUsage `json:"cpu_usage"`
		SystemUsage int64    `json:"system_cpu_usage"`
		OnlineCPUs  int      `json:"online_cpus"`
	}
	CPUUsage struct {
		Total      int64 `json:"total_usage"`
		KernelMode int64 `json:"usage_in_kernelmode"`
		UserMode   int64 `json:"usage_in_usermode"`
	}
	MemStats struct {
		Usage int64           `json:"usage"`
		Limit int64           `json:"limit"`
		Stats MemStatsDetails `json:"stats"`
	}
	MemStatsDetails struct {
		ActiveAnon            int64 `json:"active_anon"`
		ActiveFile            int64 `json:"active_file"`
		Anon                  int64 `json:"anon"`
		AnonTHP               int64 `json:"anon_thp"`
		File                  int64 `json:"file"`
		FileDirty             int64 `json:"file_dirty"`
		FileMapped            int64 `json:"file_mapped"`
		FileWriteback         int64 `json:"file_writeback"`
		InactiveAnon          int64 `json:"inactive_anon"`
		InactiveFile          int64 `json:"inactive_file"`
		KernelStack           int64 `json:"kernel_stack"`
		PgActivate            int64 `json:"pgactivate"`
		PgDeactivate          int64 `json:"pgdeactivate"`
		PgFault               int64 `json:"pgfault"`
		PgLazyFree            int64 `json:"pglazyfree"`
		PgLazyFreed           int64 `json:"pglazyfreed"`
		PgMajFault            int64 `json:"pgmajfault"`
		PgRefill              int64 `json:"pgrefill"`
		PgScan                int64 `json:"pgscan"`
		PgSteal               int64 `json:"pgsteal"`
		ShMem                 int64 `json:"shmem"`
		SLab                  int64 `json:"slab"`
		SLabReclaimable       int64 `json:"slab_reclaimable"`
		SLabUnreclaimable     int64 `json:"slab_unreclaimable"`
		Sock                  int64 `json:"sock"`
		THPCollapseAlloc      int64 `json:"thp_collapse_alloc"`
		THPFaultAlloc         int64 `json:"thp_fault_alloc"`
		Unevictable           int64 `json:"unevictable"`
		WorkingsetActivate    int64 `json:"workingset_activate"`
		WorkingSetNodeReclaim int64 `json:"workingset_nodereclaim"`
		WorkingSetRefault     int64 `json:"workingset_refault"`
	}
	NetworksStats map[string]struct {
		RXBytes int64 `json:"rx_bytes"`
		TXBytes int64 `json:"tx_bytes"`
		// "networks": {
		// "eth0": {
		//     "rx_bytes": 734887,
		//     "rx_packets": 3606,
		//     "rx_errors": 0,
		//     "rx_dropped": 0,
		//     "tx_bytes": 1773,
		//     "tx_packets": 16,
		//     "tx_errors": 0,
		//     "tx_dropped": 0
		// }
		// }
	}
)

func GetStatSummary(cli *client.Client, ctx context.Context, id string) (*SummaryStats, error) {
	reader, err := cli.ContainerStatsOneShot(ctx, id)
	if err != nil {
		return nil, err
	}
	cs, err := readContainerStats(reader.Body)
	if err != nil {
		return nil, err
	}
	var netInput, netOutput int64
	for _, v := range cs.Networks {
		netInput += v.RXBytes
		netOutput += v.TXBytes
	}
	return &SummaryStats{
		UsedMemory:      cs.Mem.Usage,
		AvailableMemory: cs.Mem.Limit,
		MemoryUsage:     float64(100) * float64(cs.Mem.Usage) / float64(cs.Mem.Limit),
		NumberCPUs:      cs.CPU.OnlineCPUs,
		CPUDelta:        cs.CPU.Usage.Total - cs.PreCPU.Usage.Total,
		SystemCPUDelta:  cs.CPU.SystemUsage - cs.PreCPU.SystemUsage,
		CPUUsage:        float64(100) * float64(cs.CPU.Usage.Total-cs.PreCPU.Usage.Total) / float64(cs.CPU.SystemUsage-cs.PreCPU.SystemUsage) * float64(cs.CPU.OnlineCPUs),
		NetInput:        netInput,
		NetOutput:       netOutput,
	}, nil
}

func readContainerStats(body io.ReadCloser) (*ContainerStats, error) {
	if body == nil {
		return nil, fmt.Errorf("empty body")
	}

	content, err := io.ReadAll(body)
	if err != nil {
		return nil, err

	}
	body.Close()
	var cs ContainerStats
	err = json.Unmarshal(content, &cs)
	return &cs, err
}

// {
//     "read": "2025-04-07T17:04:25.793723586Z",
//     "preread": "0001-01-01T00:00:00Z",
//     "pids_stats": {
//         "current": 5,
//         "limit": 14144
//     },
//     "blkio_stats": {
//         "io_service_bytes_recursive": [
//             {
//                 "major": 8,
//                 "minor": 0,
//                 "op": "read",
//                 "value": 0
//             },
//             {
//                 "major": 8,
//                 "minor": 0,
//                 "op": "write",
//                 "value": 12288
//             }
//         ],
//         "io_serviced_recursive": null,
//         "io_queue_recursive": null,
//         "io_service_time_recursive": null,
//         "io_wait_time_recursive": null,
//         "io_merged_recursive": null,
//         "io_time_recursive": null,
//         "sectors_recursive": null
//     },
//     "num_procs": 0,
//     "storage_stats": {},
//     "cpu_stats": {
//         "cpu_usage": {
//             "total_usage": 72059000,
//             "usage_in_kernelmode": 34463000,
//             "usage_in_usermode": 37596000
//         },
//         "system_cpu_usage": 609908020000000,
//         "online_cpus": 4,
//         "throttling_data": {
//             "periods": 0,
//             "throttled_periods": 0,
//             "throttled_time": 0
//         }
//     },
//     "precpu_stats": {
//         "cpu_usage": {
//             "total_usage": 0,
//             "usage_in_kernelmode": 0,
//             "usage_in_usermode": 0
//         },
//         "throttling_data": {
//             "periods": 0,
//             "throttled_periods": 0,
//             "throttled_time": 0
//         }
//     },
//     "memory_stats": {
//         "usage": 4734976,
//         "stats": {
//             "active_anon": 12288,
//             "active_file": 16384,
//             "anon": 3727360,
//             "anon_thp": 0,
//             "file": 20480,
//             "file_dirty": 0,
//             "file_mapped": 4096,
//             "file_writeback": 0,
//             "inactive_anon": 3719168,
//             "inactive_file": 0,
//             "kernel_stack": 81920,
//             "pgactivate": 0,
//             "pgdeactivate": 0,
//             "pgfault": 4405,
//             "pglazyfree": 0,
//             "pglazyfreed": 0,
//             "pgmajfault": 0,
//             "pgrefill": 0,
//             "pgscan": 6,
//             "pgsteal": 2,
//             "shmem": 4096,
//             "slab": 557848,
//             "slab_reclaimable": 229296,
//             "slab_unreclaimable": 328552,
//             "sock": 0,
//             "thp_collapse_alloc": 0,
//             "thp_fault_alloc": 0,
//             "unevictable": 0,
//             "workingset_activate": 0,
//             "workingset_nodereclaim": 0,
//             "workingset_refault": 0
//         },
//         "limit": 12443324416
//     },
//     "networks": {
//         "eth0": {
//             "rx_bytes": 734887,
//             "rx_packets": 3606,
//             "rx_errors": 0,
//             "rx_dropped": 0,
//             "tx_bytes": 1773,
//             "tx_packets": 16,
//             "tx_errors": 0,
//             "tx_dropped": 0
//         }
//     }
// }
