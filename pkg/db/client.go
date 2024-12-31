package db

import (
	"fmt"
	"sync"

	"github.com/alpacanetworks/alpamon-go/pkg/db/ent"
	_ "github.com/glebarez/go-sqlite"
)

var (
	client  *ent.Client
	once    sync.Once
	initErr error
)

func GetClient(path string) (*ent.Client, error) {
	once.Do(func() {
		var err error
		url := fmt.Sprintf("file:%s?cache=shared&__pragma=foreign_keys(1)", path)
		client, err = ent.Open("sqlite3", url)
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
