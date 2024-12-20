package db

import (
	"fmt"
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

func GetClient(path string) (*ent.Client, error) {
	once.Do(func() {
		var err error
		url := fmt.Sprintf("file:%s?cache=shared&_fk=1", path)
		client, err = ent.Open(dialect.SQLite, url)
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
