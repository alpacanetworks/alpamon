package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// MemoryPerHour holds the schema definition for theMemoryPerHour entity.
type MemoryPerHour struct {
	ent.Schema
}

// Fields of the MemoryPerHour.
func (MemoryPerHour) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.Float("peak_usage"),
		field.Float("avg_usage"),
	}
}

func (MemoryPerHour) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
