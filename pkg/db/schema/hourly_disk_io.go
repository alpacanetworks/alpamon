package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// HourlyDiskIO holds the schema definition for the HourlyDiskIO entity.
type HourlyDiskIO struct {
	ent.Schema
}

// Fields of the HourlyDiskIO.
func (HourlyDiskIO) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.String("device"),
		field.Float("peak_read_bps"),
		field.Float("peak_write_bps"),
		field.Float("avg_read_bps"),
		field.Float("avg_write_bps"),
	}
}

func (HourlyDiskIO) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
