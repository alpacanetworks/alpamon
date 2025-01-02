package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Traffic holds the schema definition for the Traffic entity.
type Traffic struct {
	ent.Schema
}

// Fields of the Traffic.
func (Traffic) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.String("name"),
		field.Float("input_pps"),
		field.Float("input_bps"),
		field.Float("output_pps"),
		field.Float("output_bps"),
	}
}

func (Traffic) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
