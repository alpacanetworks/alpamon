package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// HourlyCPUUsage holds the schema definition for the HourlyCPUUsage entity.
type HourlyCPUUsage struct {
	ent.Schema
}

// Fields of the HourlyCPUUsage.
func (HourlyCPUUsage) Fields() []ent.Field {
	return []ent.Field{
		field.Time("timestamp").Default(time.Now()),
		field.Float("peak"),
		field.Float("avg"),
	}
}

func (HourlyCPUUsage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timestamp"),
	}
}
