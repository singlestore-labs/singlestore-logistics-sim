CREATE DATABASE logistics;
USE logistics;

-- the packages table stores one row per package
CREATE TABLE packages (
    -- packageid is a UUID stored in its canonical text representation
    -- (32 hexadecimal characters and 4 hyphens)
    packageid CHAR(36) NOT NULL,

    -- marks when the package was received
    received DATETIME NOT NULL,

    -- marks when the package is expected to be delivered
    delivery_estimate DATETIME NOT NULL,

    -- marks when the package was delivered
    delivered DATETIME,

    -- origin_locationid specifies the location where the package was originally
    -- received
    origin_locationid BIGINT NOT NULL,

    -- destination_locationid specifies the package's destination location
    destination_locationid BIGINT NOT NULL,

    -- the shipping method selected
    -- standard packages are delivered using the slowest method at each point
    -- express packages are delivered using the fastest method at each point
    method ENUM ('standard', 'express') NOT NULL,

    -- lonlat is the last reported real-time position of the package
    -- this attribute does not always correspond to the locations table as it is
    -- updated while the package moves in real-time between different locations
    lonlat GEOGRAPHYPOINT NOT NULL,

    -- updated automatically keeps track of when this row was last changed
    updated DATETIME NOT NULL DEFAULT NOW() ON UPDATE NOW(),

    PRIMARY KEY (packageid),
    INDEX (lonlat),
    INDEX (updated)
);

CREATE REFERENCE TABLE locations (
    locationid BIGINT NOT NULL,

    -- each location in our distribution network is either a hub or a pickup-dropoff point
    -- a hub is usually located in larger cities and acts as both a pickup-dropoff and transit location
    -- a point only supports pickup or dropoff - it can't handle a large package volume
    kind ENUM ('hub', 'point') NOT NULL,

    -- useful metadata for queries
    city TEXT NOT NULL,
    country TEXT NOT NULL,
    city_population BIGINT NOT NULL,

    lonlat GEOGRAPHYPOINT NOT NULL,

    PRIMARY KEY (locationid),
    INDEX (lonlat)
);

-- we use this cities database to dynamically generate locations
-- cities with populations > 1,000,000 become hubs, the rest become points
LOAD DATA INFILE '/data/simplemaps/worldcities.csv'
INTO TABLE locations
FIELDS TERMINATED BY ',' ENCLOSED BY '"'
LINES TERMINATED BY '\n'
IGNORE 1 LINES
(city, @, @lat, @lon, country, @, @, @, @, @population, locationid)
SET
    -- data is a bit messy - lets assume 0 people means 100 people
    city_population = IF(@population = 0, 100, @population),
    kind = IF(@population > 1000000, "hub", "point"),
    lonlat = CONCAT('POINT(', @lon, ' ', @lat, ')');

CREATE TABLE package_transitions (
    packageid CHAR(36) NOT NULL,

    -- each package transition is assigned a strictly monotonically increasing sequence number
    seq INT NOT NULL,

    -- the location of the package where this transition occurred
    locationid BIGINT NOT NULL,

    -- the location of the next transition for this package
    -- currently only used for departure scans
    next_locationid BIGINT,

    -- when did this transition happen
    recorded DATETIME NOT NULL,

    kind ENUM (
        -- arrival scan means the package was received
        'arrival scan',
        -- departure scan means the package is enroute to another location
        'departure scan',
        -- delivered means the package was successfully delivered
        'delivered'
    ) NOT NULL,

    PRIMARY KEY (packageid, seq),
    SHARD (packageid)
);

/*
DELIMITER //

CREATE PROCEDURE process_package_states(batch QUERY(
    packageid BINARY(16) NOT NULL,
    locationid BINARY(16) NOT NULL,
    created DATETIME(6) NOT NULL,
    state TEXT NOT NULL
))
AS
BEGIN
    INSERT INTO package_states (packageid, locationid, created, state) SELECT * FROM batch;
END //

DELIMITER ;

CREATE PIPELINE package_states
AS LOAD DATA KAFKA 'redpanda/package_states'
INTO PROCEDURE process_package_states
FORMAT AVRO (
    @packageid <- packageid,
    @locationid <- locationid,
    created <- created,
    state <- state
)
SCHEMA '{
    "type": "record",
    "name": "data",
    "fields": [
        { "name": "packageid", "type": "string" },
        { "name": "locationid", "type": "string" },
        { "name": "created", "type": "string" },
        { "name": "state", "type": "string" }
    ]
}'
SET
    packageid = UNHEX(@packageid),
    locationid = UNHEX(@locationid);

START PIPELINE package_states;
*/