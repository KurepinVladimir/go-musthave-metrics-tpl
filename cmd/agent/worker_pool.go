package main

import (
	"context"
	"log"
	"sync"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/models"
)

func startWorkers(ctx context.Context, n int, jobs <-chan models.Metrics, agent *Agent) *sync.WaitGroup {
	if n < 1 {
		n = 1
	}
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case m, ok := <-jobs:
					if !ok {
						return
					}
					if err := agent.sendMetricJSON(m); err != nil {
						log.Printf("[worker %d] send error for %s: %v", id, m.ID, err)
					}
				}
			}
		}(i + 1)
	}
	return &wg
}
