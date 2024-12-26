package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DiskUsagePerHour holds the schema definition for the DiskUsagePerHour entity.
type DiskUsagePerHour struct {
	ent.Schema
}

// Fields of the DiskUsagePerHour.
func (DiskUsagePerHour) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.String("device"),
		field.Float("peak_usage"),
		field.Float("avg_usage"),
	}
}

func (DiskUsagePerHour) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
