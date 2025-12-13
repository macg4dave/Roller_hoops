package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Pool, error) {
	p, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	// Verify connectivity early.
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, err
	}

	return &Pool{pool: p}, nil
}

func (p *Pool) Close() {
	if p == nil || p.pool == nil {
		return
	}
	p.pool.Close()
}

func (p *Pool) Ping(ctx context.Context) error {
	if p == nil || p.pool == nil {
		return nil
	}
	return p.pool.Ping(ctx)
}
