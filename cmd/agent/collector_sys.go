package main

import (
	"context"
	"fmt"
	"time"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/models"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

func collectSysLoop(ctx context.Context, every time.Duration, out chan<- models.Metrics) {
	t := time.NewTicker(every)
	defer t.Stop()

	postGauge := func(id string, v float64) {
		val := v
		out <- models.Metrics{ID: id, MType: "gauge", Value: &val}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil {
				postGauge("TotalMemory", float64(vm.Total))
				postGauge("FreeMemory", float64(vm.Free))
			}
			if perc, err := cpu.PercentWithContext(ctx, 200*time.Millisecond, true); err == nil {
				for i, p := range perc {
					postGauge(fmt.Sprintf("CPUutilization%d", i+1), p)
				}
			}
		}
	}
}
