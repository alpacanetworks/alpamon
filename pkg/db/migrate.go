package db

import (
	"context"
	"fmt"
	"os"

	"ariga.io/atlas-go-sdk/atlasexec"
	"github.com/rs/zerolog/log"
)

func RunMigration(path string, ctx context.Context) error {
	workDir, err := atlasexec.NewWorkingDir(
		atlasexec.WithMigrations(
			os.DirFS("../../pkg/db/migration"),
		),
	)
	if err != nil {
		log.Error().Err(err).Msgf("failed to get migration dir: %v", err)
		return err
	}
	defer workDir.Close()

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
