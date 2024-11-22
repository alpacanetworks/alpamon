package db

import (
	"context"

	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent/migrate"
)

// TODO: Apply Versioned Migrations
func RunMigration(ctx context.Context, client *ent.Client) error {
	err := client.Schema.Create(
		ctx,
		migrate.WithDropIndex(true),
		migrate.WithDropColumn(true),
	)

	if err != nil {
		return err
	}

	return nil
}
