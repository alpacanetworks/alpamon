package db

import (
	"context"
	"fmt"
	"os"

	"ariga.io/atlas-go-sdk/atlasexec"
)

func RunMigration(path string, ctx context.Context) error {
	workDir, err := atlasexec.NewWorkingDir(
		atlasexec.WithMigrations(
			os.DirFS("../../pkg/db/migration"),
		),
	)
	if err != nil {
		return err
	}
	defer workDir.Close()

	client, err := atlasexec.NewClient(workDir.Path(), "atlas")
	if err != nil {
		return err
	}

	url := fmt.Sprintf("sqlite://%s", path)

	_, err = client.MigrateApply(ctx, &atlasexec.MigrateApplyParams{
		URL: url,
	})

	if err != nil {
		return err
	}

	return nil
}
