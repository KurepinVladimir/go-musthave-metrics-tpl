package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/models"
	_ "github.com/jackc/pgx/v5/stdlib"

	"fmt"
	"time"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/logger"
	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/pgerrors"
	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/retry"
	"go.uber.org/zap"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{
		db: db,
	}
}

var pgDelays = []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

func (p *PostgresStorage) execWithRetry(ctx context.Context, query string, args ...any) error {
	return retry.DoIf(ctx, pgDelays, func(ctx context.Context) error {
		_, err := p.db.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("%s: %w", "pg exec", err)
		}
		return nil
	}, pgerrors.IsRetriable)
}

func (p *PostgresStorage) UpdateGauge(ctx context.Context, name string, value float64) {
	if err := p.execWithRetry(ctx, `
		INSERT INTO gauge_metrics (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
	`, name, value); err != nil {
		logger.Log.Error("update gauge failed", zap.Error(err))
	}
}

func (p *PostgresStorage) UpdateCounter(ctx context.Context, name string, delta int64) {
	if err := p.execWithRetry(ctx, `
		INSERT INTO counter_metrics (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = counter_metrics.value + EXCLUDED.value
	`, name, delta); err != nil {
		logger.Log.Error("update counter failed", zap.Error(err))
	}
}

func (p *PostgresStorage) GetGauge(ctx context.Context, name string) (float64, bool) {
	var val float64
	err := p.db.QueryRowContext(ctx, `SELECT value FROM gauge_metrics WHERE name = $1`, name).Scan(&val)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false
	}
	return val, err == nil
}

func (p *PostgresStorage) GetCounter(ctx context.Context, name string) (int64, bool) {
	var val int64
	err := p.db.QueryRowContext(ctx, `SELECT value FROM counter_metrics WHERE name = $1`, name).Scan(&val)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false
	}
	return val, err == nil
}

func (p *PostgresStorage) GetAllMetrics(ctx context.Context) (map[string]float64, map[string]int64) {
	gauges := make(map[string]float64)
	counters := make(map[string]int64)

	rows, err := p.db.QueryContext(ctx, `SELECT name, value FROM gauge_metrics`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			var val float64
			if err := rows.Scan(&name, &val); err == nil {
				gauges[name] = val
			}
			if err := rows.Err(); err != nil {
				return nil, nil
			}
		}
	}

	rows, err = p.db.QueryContext(ctx, `SELECT name, value FROM counter_metrics`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			var val int64
			if err := rows.Scan(&name, &val); err == nil {
				counters[name] = val
			}
			if err := rows.Err(); err != nil {
				return nil, nil
			}
		}
	}

	return gauges, counters
}

func (p *PostgresStorage) UpdateBatch(ctx context.Context, batch []models.Metrics) error {
	tx, err := p.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, m := range batch {
		switch m.MType {
		case "gauge":
			if m.Value == nil {
				continue
			}
			_, err = tx.ExecContext(ctx, `
				INSERT INTO gauge_metrics (name, value)
				VALUES ($1, $2)
				ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
			`, m.ID, *m.Value)
		case "counter":
			if m.Delta == nil {
				continue
			}
			_, err = tx.ExecContext(ctx, `
				INSERT INTO counter_metrics (name, value)
				VALUES ($1, $2)
				ON CONFLICT (name) DO UPDATE SET value = counter_metrics.value + EXCLUDED.value
			`, m.ID, *m.Delta)
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
