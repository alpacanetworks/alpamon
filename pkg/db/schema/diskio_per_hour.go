package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DiskIOPerHour holds the schema definition for the DiskIOPerHour entity.
type DiskIOPerHour struct {
	ent.Schema
}

// Fields of the DiskIOPerHour.
func (DiskIOPerHour) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.String("device"),
		field.Int64("peak_read_bytes"),
		field.Int64("peak_write_bytes"),
		field.Int64("avg_read_bytes"),
		field.Int64("avg_write_bytes"),
	}
}

func (DiskIOPerHour) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
