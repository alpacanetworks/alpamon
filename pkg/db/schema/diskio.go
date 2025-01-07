package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DiskIO holds the schema definition for the DiskIO entity.
type DiskIO struct {
	ent.Schema
}

// Fields of the DiskIO.
func (DiskIO) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.String("device"),
		field.Float("read_bps"),
		field.Float("write_bps"),
	}
}

func (DiskIO) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
