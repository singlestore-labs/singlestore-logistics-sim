# singlestore-logistics-sim

Large scale logistics demo using [SingleStore](https://singlestore.com) and [Redpanda](https://vectorized.io/).

## Usage

1. [Sign up](https://msql.co/2E8aBa2) for a free SingleStore license.
This allows you to run up to 4 nodes up to 32 gigs each for free. Grab your license key from [SingleStore portal](https://msql.co/3fZoxjO) and set it as an environment variable.

2. Run the simulation with `make`.

```bash
export SINGLESTORE_LICENSE="<<singlestore license>>"
make
```

## Simulator

The [simulator](simulator) is a go program which generates package histories and writes them into Redpanda topics.

There are three topics:
 - packages
 - transitions

## packages topic

The packages topic contains a record per package. The record is written when we receive the package in question.

**Avro schema**:

```json
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
```

## transitions topic

The transitions topic is written to whenever a package changes states. A normal package goes through the following transitions during it's lifetime:

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
```

## Interesting Queries

### Show the full history of a single package

```sql
SELECT
    pt.seq,
    pt.kind,
    current_loc.city,
    current_loc.country,
    GEOGRAPHY_DISTANCE(current_loc.lonlat, destination.lonlat) / 1000 AS distance_to_destination,
    pt.recorded
FROM package_transitions pt
INNER JOIN locations current_loc ON pt.locationid = current_loc.locationid
INNER JOIN packages p ON pt.packageid = p.packageid
INNER JOIN locations destination ON p.destination_locationid = destination.locationid
WHERE pt.packageid = '516aa045-d8df-4363-b250-da335df82269'
ORDER BY seq DESC;
```