# singlestore-logistics-sim

Large scale logistics demo using [SingleStore](https://singlestore.com) and [Redpanda](https://vectorized.io/).

## Usage

1. [Sign up](https://msql.co/2E8aBa2) for a free SingleStore license.
This allows you to run up to 4 nodes up to 32 gigs each for free. Grab your license key from [SingleStore portal](https://msql.co/3fZoxjO) and set it as an environment variable.

2. Run the simulation with `docker-compose up`.

```bash
export SINGLESTORE_LICENSE="<<singlestore license>>"
docker-compose up
```

## Simulator

The [simulator](simulator) is a go program which generates package histories and writes them into Redpanda topics.

There are three topics:
 - packages
 - package_states
 - tracking

## packages topic

The packages topic contains a record per package. The record is written when we receive the package in question.

**Avro schema**:

```json
{
    "type": "record",
    "name": "Package",
    "fields": [
        { "name": "PackageID", "type": "string", "logicalType": "uuid" },
        { "name": "Received", "type": "long", "logicalType": "timestamp-millis" },
        { "name": "DeliveryEstimate", "type": "long", "logicalType": "timestamp-millis" },
        { "name": "OriginLocationID", "type": "long" },
        { "name": "DestinationLocationID", "type": "long" },
        { "name": "Method", "type": "enum", "symbols": [
            "standard", "express"
        ] },
    ]
}
```

## package_transitions topic

The package_transitions topic is written to whenever a package changes states. A normal package goes through the following transitions during it's lifetime:

1. arrival scan - the package has been received
2. departure scan - the package has been scanned and put in transit to another location
    * arrival scan and departure scan can occur multiple times as the package moves through our global logistics network
3. delivered - the package has been delivered

*Note: We don't currently model last-mile delivery, perhaps we will add that in a future iteration.*

**Avro schema**:

```json
{
    "type": "record",
    "name": "PackageTransition",
    "fields": [
        { "name": "PackageID", "type": "string", "logicalType": "uuid" },
        { "name": "Seq", "type": "int" },
        { "name": "LocationID", "type": "long" },
        { "name": "NextLocationID", "type": ["null", "long"] },
        { "name": "Recorded", "type": "long", "logicalType": "timestamp-millis" },
        { "name": "Kind", "type": "enum", "symbols": [
            "arrival scan", "departure scan", "delivered"
        ] }
    ]
}
```

## tracking topic

The tracking topic is written to as packages move in real time.

**Avro schema**:

```json
{
    "type": "record",
    "name": "Track",
    "fields": [
        { "name": "PackageID", "type": "string", "logicalType": "uuid" },
        { "name": "Recorded", "type": "long", "logicalType": "timestamp-millis" },
        { "name": "Longitude", "type": "double" },
        { "name": "Latitude", "type": "double" },
    ]
```