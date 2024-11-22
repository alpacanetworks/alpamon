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
		field.Int64("input_pkts"),
		field.Int64("input_bytes"),
		field.Int64("output_pkts"),
		field.Int64("output_bytes"),
	}
}

func (Traffic) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
