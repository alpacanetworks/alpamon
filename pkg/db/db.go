package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/glebarez/go-sqlite"
	"github.com/rs/zerolog/log"
)

const (
	dbDir      = "/var/lib/alpamon"
	dbFileName = "alpamon.db"
)

func InitDB(ctx context.Context) *ent.Client {
	fileName := fmt.Sprintf("%s/%s", dbDir, dbFileName)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		fileName, _ = filepath.Abs(dbFileName)
	}

	dbFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0750)
	if err != nil {
		log.Error().Err(err).Msgf("failed to open db file: %v", err)
		_, _ = fmt.Fprintf(os.Stderr, "Failed to open db file: %v\n", err)
		os.Exit(1)
	}

	sql.Register("sqlite3", &sqlite.Driver{})

	err = RunMigration(dbFile.Name(), ctx)
	if err != nil {
		log.Error().Err(err).Msgf("failed to migrate db: %v\n", err)
		os.Exit(1)
	}

	client, err := GetClient(dbFile.Name())
	if err != nil {
		log.Error().Err(err).Msgf("failed to get db client: %v\n", err)
		os.Exit(1)
	}

	return client
}
