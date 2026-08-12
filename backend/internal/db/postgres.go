package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresDB обёртка над пулом подключений PostgreSQL
type PostgresDB struct {
	Pool *pgxpool.Pool
}

// NewPostgresDB создаёт подключение к PostgreSQL
func NewPostgresDB(databaseURL string) (*PostgresDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Создаём пул подключений
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания пула: %w", err)
	}

	// Проверяем подключение
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ошибка ping: %w", err)
	}

	log.Println("✅ Подключение к PostgreSQL установлено")
	return &PostgresDB{Pool: pool}, nil
}

// Close закрывает пул подключений
func (db *PostgresDB) Close() {
	db.Pool.Close()
	log.Println("🔒 Подключение к PostgreSQL закрыто")
}

// Health проверяет доступность БД
func (db *PostgresDB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}