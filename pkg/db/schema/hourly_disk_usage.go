package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// HourlyDiskUsage holds the schema definition for the HourlyDiskUsage entity.
type HourlyDiskUsage struct {
	ent.Schema
}

// Fields of the HourlyDiskUsage.
func (HourlyDiskUsage) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.String("device"),
		field.Float("peak"),
		field.Float("avg"),
		field.Int64("total").Default(0),
		field.Int64("free").Default(0),
		field.Int64("used").Default(0),
	}
}

func (HourlyDiskUsage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
