package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// MemoryPerHour holds the schema definition for the MemoryPerHour entity.
type MemoryPerHour struct {
	ent.Schema
}

// Fields of the MemoryPerHour.
func (MemoryPerHour) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.Float("peak"),
		field.Float("avg"),
	}
}

func (MemoryPerHour) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
