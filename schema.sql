-- General schema notes:
--  * BINARY(16) ids are UUID's stored as bytes for performance

create database logistics;
use logistics;

-- the packages table stores one row per package
create table packages (
    packageid BINARY(16) NOT NULL,

    -- marks when the package was registered in our system
    created DATETIME NOT NULL DEFAULT NOW(),

    -- marks when the package is expected to be delivered
    delivery_estimate DATETIME NOT NULL,

    -- marks when the package was physically received
    received DATETIME,

    -- marks when the package was physically delivered
    delivered DATETIME,

    -- source_locationid specifies the package's source location
    source_locationid BINARY(16) NOT NULL,

    -- dropoff_locationid specifies the location where the package was originally received
    -- this column is null if the package has not yet been received
    dropoff_locationid BINARY(16),

    -- destination_locationid specifies the original destination location of the package
    destination_locationid BINARY(16) NOT NULL,

    -- return_locationid specifies where the package should go if it is returned
    return_locationid BINARY(16) NOT NULL,

    -- the shipping method selected
    method ENUM ('standard', 'express', 'priority') NOT NULL,

    -- the type of packaging used
    packaging ENUM (
        'custom', 'small box', 'medium box', 'large box',
        'pallet', 'tube', 'letter envelope', 'document envelope'
    ) NOT NULL,

    -- the size of the package stored as a packed vector with the elements (length, width, height)
    -- each dimension is measured in inches
    dimensions BINARY(12) NOT NULL,

    -- the weight of the package measured in pounds
    weight FLOAT NOT NULL,

    -- lonlat is the last reported location of the package
    lonlat GEOGRAPHYPOINT NOT NULL,

    -- updated keeps track of when this row was last changed
    updated DATETIME NOT NULL DEFAULT NOW() ON UPDATE NOW(),

    PRIMARY KEY (packageid)
);

create table locations (
    locationid BINARY(16) NOT NULL,

    kind ENUM ('distribution center', 'store', 'business', 'personal') NOT NULL,

    lonlat GEOGRAPHYPOINT NOT NULL,

    -- the label is stored as a single formatted string
    label TEXT NOT NULL,

    PRIMARY KEY (locationid)
);

create table package_states (
    packageid BINARY(16) NOT NULL,
    stateid BIGINT NOT NULL AUTO_INCREMENT,

    -- the location of the package when it entered this state
    locationid BINARY(16) NOT NULL,

    -- marks when the package entered this state
    created DATETIME(6) NOT NULL,

    state ENUM (
        -- arrival scan means the package was received at a drop off location or distribution center
        'arrival scan',
        -- in transit means the package is travelling to a distribution center
        'in transit',
        -- out for delivery means the package is on a last-mile delivery vehicle
        'out for delivery',
        -- delivered means the package was successfully delivered to the destination
        'delivered'
    ) NOT NULL,

    PRIMARY KEY (packageid, stateid),
    SHARD (packageid)
);

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