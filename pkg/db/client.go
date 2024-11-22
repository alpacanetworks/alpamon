package db

import (
	"sync"

	"entgo.io/ent/dialect"
	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	_ "github.com/mattn/go-sqlite3"
)

var (
	client  *ent.Client
	once    sync.Once
	initErr error
)

func GetClient() (*ent.Client, error) {
	once.Do(func() {
		var err error
		client, err = ent.Open(dialect.SQLite, "file:./metrics.db?cache=shared&_fk=1")
		if err != nil {
			initErr = err
			client = nil
		}
	})
	return client, initErr
}

func Close() error {
	if client != nil {
		return client.Close()
	}
	return nil
}
