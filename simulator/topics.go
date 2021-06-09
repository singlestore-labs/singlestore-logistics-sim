package simulator

import (
	"simulator/enum"
	"time"

	"github.com/hamba/avro"
)

type Topics interface {
	WritePackage(p *Package) error
	WriteTransition(now time.Time, transition enum.TransitionKind, t *Tracker) error
	WriteLocation(now time.Time, t *Tracker) error
	Close() error
}

var (
	packageSchema = avro.MustParse(`
		{
			"type": "record",
			"name": "Package",
			"fields": [
				{ "name": "PackageID", "type": { "type": "string", "logicalType": "uuid" } },
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

	locationSchema = avro.MustParse(`
		{
			"type": "record",
			"name": "Track",
			"fields": [
				{ "name": "PackageID", "type": { "type": "string", "logicalType": "uuid" } },
				{ "name": "Recorded", "type": { "type": "long", "logicalType": "timestamp-millis" } },
				{ "name": "Position", "type": "string" }
			]
		}
	`)
)

type Redpanda struct {
	producer *SaramaProducer

	packageEncoder    *avro.Encoder
	transitionEncoder *avro.Encoder
	locationEncoder   *avro.Encoder
}

var _ Topics = &Redpanda{}

func NewRedpanda(config TopicsConfig) (*Redpanda, error) {
	producer, err := NewSaramaProducer(config.Brokers)
	if err != nil {
		return nil, err
	}

	packageEncoder := avro.NewEncoderForSchema(packageSchema, producer.TopicWriter("packages"))
	transitionEncoder := avro.NewEncoderForSchema(transitionSchema, producer.TopicWriter("transitions"))
	locationEncoder := avro.NewEncoderForSchema(locationSchema, producer.TopicWriter("locations"))

	return &Redpanda{
		producer: producer,

		packageEncoder:    packageEncoder,
		transitionEncoder: transitionEncoder,
		locationEncoder:   locationEncoder,
	}, nil
}

func (r *Redpanda) WritePackage(p *Package) error {
	return r.packageEncoder.Encode(p)
}

func (r *Redpanda) WriteTransition(now time.Time, transition enum.TransitionKind, t *Tracker) error {
	return r.transitionEncoder.Encode(&Transition{
		PackageID:      t.PackageID,
		Seq:            t.Seq,
		LocationID:     t.LastLocationID,
		NextLocationID: t.NextLocationID,
		Recorded:       now,
		Kind:           transition,
	})
}
func (r *Redpanda) WriteLocation(now time.Time, t *Tracker) error {
	return r.locationEncoder.Encode(&LocationRecord{
		PackageID: t.PackageID,
		Recorded:  now,
		Position:  AvroPoint(t.Position),
	})
}

func (r *Redpanda) Close() error {
	return r.producer.Close()
}
