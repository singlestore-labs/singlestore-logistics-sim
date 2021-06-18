package simulator

import (
	"simulator/enum"
	"time"

	"github.com/hamba/avro"
)

var (
	packageSchema = avro.MustParse(`
		{
			"type": "record",
			"name": "Package",
			"fields": [
				{ "name": "PackageID", "type": { "type": "string", "logicalType": "uuid" } },
				{ "name": "SimulatorID", "type": "string" },
				{ "name": "Received", "type": { "type": "long", "logicalType": "timestamp-millis" } },
				{ "name": "DeliveryEstimate", "type": { "type": "long", "logicalType": "timestamp-millis" } },
				{ "name": "OriginLocationID", "type": "long" },
				{ "name": "DestinationLocationID", "type": "long" },
				{ "name": "Method", "type": { "name": "Method", "type": "enum", "symbols": [
					"standard", "express"
				] } }
			]
		}
	`)

	transitionSchema = avro.MustParse(`
		{
			"type": "record",
			"name": "PackageTransition",
			"fields": [
				{ "name": "PackageID", "type": { "type": "string", "logicalType": "uuid" } },
				{ "name": "Seq", "type": "int" },
				{ "name": "LocationID", "type": "long" },
				{ "name": "NextLocationID", "type": ["null", "long"] },
				{ "name": "Recorded", "type": { "type": "long", "logicalType": "timestamp-millis" } },
				{ "name": "Kind", "type": { "name": "Kind", "type": "enum", "symbols": [
					"arrival_scan", "departure_scan", "delivered"
				] } }
			]
		}
	`)
)

type Topics struct {
	producer Producer

	packageEncoder    *avro.Encoder
	transitionEncoder *avro.Encoder
}

func NewTopics(producer Producer) *Topics {
	return &Topics{
		producer: producer,

		packageEncoder:    avro.NewEncoderForSchema(packageSchema, producer.TopicWriter("packages")),
		transitionEncoder: avro.NewEncoderForSchema(transitionSchema, producer.TopicWriter("transitions")),
	}
}

func (r *Topics) WritePackage(p *Package) error {
	return r.packageEncoder.Encode(p)
}

func (r *Topics) WriteTransition(now time.Time, transition enum.TransitionKind, t *Tracker) error {
	return r.transitionEncoder.Encode(&Transition{
		PackageID:      t.PackageID,
		Seq:            t.Seq,
		LocationID:     t.LastLocationID,
		NextLocationID: t.NextLocationID,
		Recorded:       now,
		Kind:           transition,
	})
}
