package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Memory holds the schema definition for the Memory entity.
type Memory struct {
	ent.Schema
}

// Fields of the Memory.
func (Memory) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.Float("usage"),
	}
}

func (Memory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
