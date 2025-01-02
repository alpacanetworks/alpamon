package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CPU holds the schema definition for the CPU entity.
type CPU struct {
	ent.Schema
}

// Fields of the CPU.
func (CPU) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.Float("usage"),
	}
}

func (CPU) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
