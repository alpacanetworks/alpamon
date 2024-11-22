package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TrafficPerHour holds the schema definition for the TrafficPerHour entity.
type TrafficPerHour struct {
	ent.Schema
}

// Fields of the TrafficPerHour.
func (TrafficPerHour) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.String("name"),
		field.Int64("peak_input_pkts"),
		field.Int64("peak_input_bytes"),
		field.Int64("peak_output_pkts"),
		field.Int64("peak_output_bytes"),
		field.Int64("avg_input_pkts"),
		field.Int64("avg_input_bytes"),
		field.Int64("avg_output_pkts"),
		field.Int64("avg_output_bytes"),
	}
}

func (TrafficPerHour) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
