package repository

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// DBPoolConfig holds configuration parameters for the database connection pool.
type DBPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultDBPoolConfig returns production-grade defaults for PostgreSQL.
func DefaultDBPoolConfig() DBPoolConfig {
	return DBPoolConfig{
		MaxOpenConns:    50,
		MaxIdleConns:    15,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}
}

// DBPoolConfigFromEnv reads pool settings from environment variables with fallback to defaults.
func DBPoolConfigFromEnv() DBPoolConfig {
	cfg := DefaultDBPoolConfig()

	if val := os.Getenv("DB_MAX_OPEN_CONNS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			cfg.MaxOpenConns = n
		}
	}

	if val := os.Getenv("DB_MAX_IDLE_CONNS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			cfg.MaxIdleConns = n
		}
	}

	if val := os.Getenv("DB_CONN_MAX_LIFETIME"); val != "" {
		if d, err := time.ParseDuration(val); err == nil && d > 0 {
			cfg.ConnMaxLifetime = d
		} else if m, err := strconv.Atoi(val); err == nil && m > 0 {
			cfg.ConnMaxLifetime = time.Duration(m) * time.Minute
		}
	}

	if val := os.Getenv("DB_CONN_MAX_IDLE_TIME"); val != "" {
		if d, err := time.ParseDuration(val); err == nil && d > 0 {
			cfg.ConnMaxIdleTime = d
		} else if m, err := strconv.Atoi(val); err == nil && m > 0 {
			cfg.ConnMaxIdleTime = time.Duration(m) * time.Minute
		}
	}

	// Safety check: idle connections should never exceed max open connections
	if cfg.MaxIdleConns > cfg.MaxOpenConns {
		cfg.MaxIdleConns = cfg.MaxOpenConns
	}

	return cfg
}

// ConfigureDBPool applies pool limits and lifetime configurations to a GORM database instance.
func ConfigureDBPool(db *gorm.DB, cfg DBPoolConfig) error {
	if db == nil {
		return fmt.Errorf("gorm.DB instance is nil")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm: %w", err)
	}

	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = 50
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 15
	}
	if cfg.ConnMaxLifetime <= 0 {
		cfg.ConnMaxLifetime = 30 * time.Minute
	}
	if cfg.ConnMaxIdleTime <= 0 {
		cfg.ConnMaxIdleTime = 10 * time.Minute
	}

	if cfg.MaxIdleConns > cfg.MaxOpenConns {
		cfg.MaxIdleConns = cfg.MaxOpenConns
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	slog.Info("Database connection pool successfully configured",
		slog.Int("max_open_conns", cfg.MaxOpenConns),
		slog.Int("max_idle_conns", cfg.MaxIdleConns),
		slog.Duration("conn_max_lifetime", cfg.ConnMaxLifetime),
		slog.Duration("conn_max_idle_time", cfg.ConnMaxIdleTime),
	)

	return nil
}

// GetDBStats returns current database pool statistics from GORM.
func GetDBStats(db *gorm.DB) (sql.DBStats, error) {
	if db == nil {
		return sql.DBStats{}, fmt.Errorf("gorm.DB instance is nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return sql.DBStats{}, fmt.Errorf("failed to get sql.DB from gorm: %w", err)
	}
	return sqlDB.Stats(), nil
}

