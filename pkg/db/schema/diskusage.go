package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DiskUsage holds the schema definition for the DiskUsage entity.
type DiskUsage struct {
	ent.Schema
}

// Fields of the DiskUsage.
func (DiskUsage) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.String("device"),
		field.Float("usage"),
		field.Int64("total"),
		field.Int64("free"),
		field.Int64("used"),
	}
}

func (DiskUsage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
