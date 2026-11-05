package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestDefaultDBPoolConfig(t *testing.T) {
	cfg := DefaultDBPoolConfig()
	assert.Equal(t, 50, cfg.MaxOpenConns)
	assert.Equal(t, 15, cfg.MaxIdleConns)
	assert.Equal(t, 30*time.Minute, cfg.ConnMaxLifetime)
	assert.Equal(t, 10*time.Minute, cfg.ConnMaxIdleTime)
}

func TestDBPoolConfigFromEnv_DefaultsWhenEmpty(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "")
	t.Setenv("DB_MAX_IDLE_CONNS", "")
	t.Setenv("DB_CONN_MAX_LIFETIME", "")
	t.Setenv("DB_CONN_MAX_IDLE_TIME", "")

	cfg := DBPoolConfigFromEnv()
	assert.Equal(t, 50, cfg.MaxOpenConns)
	assert.Equal(t, 15, cfg.MaxIdleConns)
	assert.Equal(t, 30*time.Minute, cfg.ConnMaxLifetime)
	assert.Equal(t, 10*time.Minute, cfg.ConnMaxIdleTime)
}

func TestDBPoolConfigFromEnv_CustomValues(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "100")
	t.Setenv("DB_MAX_IDLE_CONNS", "25")
	t.Setenv("DB_CONN_MAX_LIFETIME", "45m")
	t.Setenv("DB_CONN_MAX_IDLE_TIME", "15m")

	cfg := DBPoolConfigFromEnv()
	assert.Equal(t, 100, cfg.MaxOpenConns)
	assert.Equal(t, 25, cfg.MaxIdleConns)
	assert.Equal(t, 45*time.Minute, cfg.ConnMaxLifetime)
	assert.Equal(t, 15*time.Minute, cfg.ConnMaxIdleTime)
}

func TestDBPoolConfigFromEnv_NumericMinutesFallback(t *testing.T) {
	t.Setenv("DB_CONN_MAX_LIFETIME", "60")
	t.Setenv("DB_CONN_MAX_IDLE_TIME", "20")

	cfg := DBPoolConfigFromEnv()
	assert.Equal(t, 60*time.Minute, cfg.ConnMaxLifetime)
	assert.Equal(t, 20*time.Minute, cfg.ConnMaxIdleTime)
}

func TestDBPoolConfigFromEnv_ClampsIdleToMaxOpen(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "10")
	t.Setenv("DB_MAX_IDLE_CONNS", "30") // Exceeds open limit

	cfg := DBPoolConfigFromEnv()
	assert.Equal(t, 10, cfg.MaxOpenConns)
	assert.Equal(t, 10, cfg.MaxIdleConns, "MaxIdleConns must be clamped to MaxOpenConns")
}

func TestConfigureDBPool_NilDB_ReturnsError(t *testing.T) {
	err := ConfigureDBPool(nil, DefaultDBPoolConfig())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gorm.DB instance is nil")

	_, statsErr := GetDBStats(nil)
	assert.Error(t, statsErr)
}

func TestConfigureDBPool_AppliesSettingsToUnderlyingDB(t *testing.T) {
	// Create an un-dialed sql.DB using the postgres driver
	sqlDB, err := sql.Open("pgx", "postgres://user:pass@localhost:5432/dbname?sslmode=disable")
	if err != nil {
		// Fallback if pgx is not directly registered as sql driver name
		sqlDB, err = sql.Open("postgres", "postgres://user:pass@localhost:5432/dbname?sslmode=disable")
	}
	assert.NoError(t, err)
	defer sqlDB.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                sqlDB,
		WithoutQuotingCheck: true,
	}), &gorm.Config{
		DisableAutomaticPing: true,
	})
	assert.NoError(t, err)

	cfg := DBPoolConfig{
		MaxOpenConns:    75,
		MaxIdleConns:    20,
		ConnMaxLifetime: 25 * time.Minute,
		ConnMaxIdleTime: 8 * time.Minute,
	}

	err = ConfigureDBPool(gormDB, cfg)
	assert.NoError(t, err)

	stats, err := GetDBStats(gormDB)
	assert.NoError(t, err)
	assert.Equal(t, 75, stats.MaxOpenConnections)
}
