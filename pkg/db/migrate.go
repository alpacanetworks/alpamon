package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"ariga.io/atlas-go-sdk/atlasexec"
	"github.com/rs/zerolog/log"
)

//go:embed migration/*
var migrations embed.FS

func RunMigration(path string, ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		log.Error().Err(err).Msgf("context cancelled before migration: %v", err)
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		migrationFS, err := getMigrationDir()
		if err != nil {
			log.Error().Err(err).Msg("failed to get migration filesystem")
			return err
		}

		workDir, err := atlasexec.NewWorkingDir(
			atlasexec.WithMigrations(
				migrationFS,
			),
		)
		if err != nil {
			log.Error().Err(err).Msgf("failed to open migration dir: %v", err)
			return err
		}
		defer func() { _ = workDir.Close() }()

		client, err := atlasexec.NewClient(workDir.Path(), "atlas")
		if err != nil {
			log.Error().Err(err).Msgf("failed to get atlas client: %v", err)
			return err
		}

		url := fmt.Sprintf("sqlite://%s", path)

		_, err = client.MigrateApply(ctx, &atlasexec.MigrateApplyParams{
			URL: url,
		})

		if err != nil {
			log.Error().Err(err).Msgf("failed to migrate db: %v", err)
			return err
		}

		return nil
	}
}

func getMigrationDir() (fs.FS, error) {
	migrationFS, err := fs.Sub(migrations, "migration")
	if err != nil {
		return nil, err
	}

	return migrationFS, nil
}
