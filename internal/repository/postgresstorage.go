package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/KurepinVladimir/go-musthave-metrics-tpl.git/internal/models"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{
		db: db,
	}
}

func (p *PostgresStorage) UpdateGauge(ctx context.Context, name string, value float64) {
	_, _ = p.db.ExecContext(ctx, `
		INSERT INTO gauge_metrics (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
	`, name, value)
}

func (p *PostgresStorage) UpdateCounter(ctx context.Context, name string, value int64) {
	_, _ = p.db.ExecContext(ctx, `
		INSERT INTO counter_metrics (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = counter_metrics.value + EXCLUDED.value
	`, name, value)
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
