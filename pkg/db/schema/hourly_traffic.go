package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// HourlyTraffic holds the schema definition for the HourlyTraffic entity.
type HourlyTraffic struct {
	ent.Schema
}

// Fields of the HourlyTraffic.
func (HourlyTraffic) Fields() []ent.Field {
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

func (HourlyTraffic) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
