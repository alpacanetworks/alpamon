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
		field.Float("peak_input_pps"),
		field.Float("peak_input_bps"),
		field.Float("peak_output_pps"),
		field.Float("peak_output_bps"),
		field.Float("avg_input_pps"),
		field.Float("avg_input_bps"),
		field.Float("avg_output_pps"),
		field.Float("avg_output_bps"),
	}
}

func (TrafficPerHour) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
