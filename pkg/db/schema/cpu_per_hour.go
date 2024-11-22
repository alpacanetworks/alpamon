package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CPUPerHour holds the schema definition for the CPUPerHour entity.
type CPUPerHour struct {
	ent.Schema
}

// Fields of the CPUPerHour.
func (CPUPerHour) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.Float("peak_usage"),
		field.Float("avg_usage"),
	}
}

func (CPUPerHour) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
