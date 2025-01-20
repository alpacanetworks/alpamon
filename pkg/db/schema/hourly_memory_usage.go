package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// HourlyMemoryUsage holds the schema definition for the HourlyMemoryUsage entity.
type HourlyMemoryUsage struct {
	ent.Schema
}

// Fields of the HourlyMemoryUsage.
func (HourlyMemoryUsage) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.Float("peak"),
		field.Float("avg"),
	}
}

func (HourlyMemoryUsage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
