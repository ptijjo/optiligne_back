-- Optiligne schema (PostgreSQL + PostGIS). Dev : models.AutoMigrate. Prod : appliquer ce SQL.

CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS feed_versions (
    id            VARCHAR(32) PRIMARY KEY,
    checksum      VARCHAR(128) UNIQUE NOT NULL,
    publisher     VARCHAR(255),
    feed_version  VARCHAR(64),
    start_date    VARCHAR(8),
    end_date      VARCHAR(8),
    active        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_feed_versions_active ON feed_versions (active);

CREATE TABLE IF NOT EXISTS agencies (
    id              VARCHAR(32) PRIMARY KEY,
    feed_version_id VARCHAR(32) NOT NULL,
    agency_id       VARCHAR(64) NOT NULL,
    name            VARCHAR(255),
    timezone        VARCHAR(64),
    UNIQUE (feed_version_id, agency_id)
);

CREATE TABLE IF NOT EXISTS routes (
    id              VARCHAR(32) PRIMARY KEY,
    feed_version_id VARCHAR(32) NOT NULL,
    route_id        VARCHAR(64) NOT NULL,
    agency_id       VARCHAR(64),
    short_name      VARCHAR(64),
    long_name       VARCHAR(255),
    route_type      INTEGER,
    UNIQUE (feed_version_id, route_id)
);
CREATE INDEX IF NOT EXISTS idx_routes_short_name ON routes (short_name);
CREATE INDEX IF NOT EXISTS idx_routes_route_type ON routes (route_type);

CREATE TABLE IF NOT EXISTS trips (
    id              VARCHAR(32) PRIMARY KEY,
    feed_version_id VARCHAR(32) NOT NULL,
    trip_id         VARCHAR(128) NOT NULL,
    route_id        VARCHAR(64),
    service_id      VARCHAR(128),
    shape_id        VARCHAR(128),
    headsign        VARCHAR(255),
    direction_id    INTEGER,
    UNIQUE (feed_version_id, trip_id)
);
CREATE INDEX IF NOT EXISTS idx_trips_route_id ON trips (route_id);
CREATE INDEX IF NOT EXISTS idx_trips_service_id ON trips (service_id);
CREATE INDEX IF NOT EXISTS idx_trips_shape_id ON trips (shape_id);

CREATE TABLE IF NOT EXISTS stops (
    id              VARCHAR(32) PRIMARY KEY,
    feed_version_id VARCHAR(32) NOT NULL,
    stop_id         VARCHAR(64) NOT NULL,
    name            VARCHAR(255),
    lat             DOUBLE PRECISION NOT NULL,
    lon             DOUBLE PRECISION NOT NULL,
    UNIQUE (feed_version_id, stop_id)
);

CREATE TABLE IF NOT EXISTS stop_times (
    id              VARCHAR(32) PRIMARY KEY,
    feed_version_id VARCHAR(32) NOT NULL,
    trip_id         VARCHAR(128) NOT NULL,
    stop_id         VARCHAR(64),
    stop_sequence   INTEGER NOT NULL,
    arrival_sec     INTEGER,
    departure_sec   INTEGER,
    shape_dist      DOUBLE PRECISION,
    UNIQUE (feed_version_id, trip_id, stop_sequence)
);
CREATE INDEX IF NOT EXISTS idx_stop_times_stop_id ON stop_times (stop_id);

CREATE TABLE IF NOT EXISTS calendars (
    id              VARCHAR(32) PRIMARY KEY,
    feed_version_id VARCHAR(32) NOT NULL,
    service_id      VARCHAR(128) NOT NULL,
    monday          BOOLEAN,
    tuesday         BOOLEAN,
    wednesday       BOOLEAN,
    thursday        BOOLEAN,
    friday          BOOLEAN,
    saturday        BOOLEAN,
    sunday          BOOLEAN,
    start_date      VARCHAR(8),
    end_date        VARCHAR(8),
    UNIQUE (feed_version_id, service_id)
);

CREATE TABLE IF NOT EXISTS calendar_dates (
    id              VARCHAR(32) PRIMARY KEY,
    feed_version_id VARCHAR(32) NOT NULL,
    service_id      VARCHAR(128),
    date            VARCHAR(8),
    exception_type  INTEGER
);
CREATE INDEX IF NOT EXISTS idx_calendar_dates_feed ON calendar_dates (feed_version_id);
CREATE INDEX IF NOT EXISTS idx_calendar_dates_service ON calendar_dates (service_id);
CREATE INDEX IF NOT EXISTS idx_calendar_dates_date ON calendar_dates (date);

CREATE TABLE IF NOT EXISTS shapes (
    id              VARCHAR(32) PRIMARY KEY,
    feed_version_id VARCHAR(32) NOT NULL,
    shape_id        VARCHAR(128) NOT NULL,
    geom            geometry(LineString, 4326),
    UNIQUE (feed_version_id, shape_id)
);
CREATE INDEX IF NOT EXISTS idx_shapes_geom ON shapes USING GIST (geom);

CREATE TABLE IF NOT EXISTS stop_fracs (
    id              VARCHAR(32) PRIMARY KEY,
    feed_version_id VARCHAR(32) NOT NULL,
    trip_id         VARCHAR(128) NOT NULL,
    stop_id         VARCHAR(64),
    stop_sequence   INTEGER NOT NULL,
    frac            DOUBLE PRECISION NOT NULL,
    arrival_sec     INTEGER,
    stop_name       VARCHAR(255),
    UNIQUE (feed_version_id, trip_id, stop_sequence)
);

CREATE TABLE IF NOT EXISTS operators (
    id   VARCHAR(32) PRIMARY KEY,
    code VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS depots (
    id          VARCHAR(32) PRIMARY KEY,
    code        VARCHAR(64) NOT NULL,
    operator_id VARCHAR(32) NOT NULL,
    name        VARCHAR(255),
    UNIQUE (code, operator_id)
);

CREATE TABLE IF NOT EXISTS route_assignments (
    id              VARCHAR(32) PRIMARY KEY,
    feed_version_id VARCHAR(32) NOT NULL,
    operator_id     VARCHAR(32) NOT NULL,
    depot_id        VARCHAR(32) NOT NULL,
    route_id        VARCHAR(64) NOT NULL,
    UNIQUE (feed_version_id, depot_id, route_id)
);
CREATE INDEX IF NOT EXISTS idx_route_assignments_operator ON route_assignments (operator_id);
