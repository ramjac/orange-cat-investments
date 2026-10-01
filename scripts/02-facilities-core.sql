-- Standard Small Business Facilities & Inventory Schema
CREATE SCHEMA IF NOT EXISTS facilities;

CREATE TABLE IF NOT EXISTS facilities.location_zones (
    zone_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    name VARCHAR(100) NOT NULL UNIQUE,
    building VARCHAR(100) NOT NULL,
    floor_level INT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS facilities.hardware_assets (
    asset_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    serial_number VARCHAR(100) NOT NULL UNIQUE,
    asset_type VARCHAR(50) NOT NULL,
    model VARCHAR(100) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'active',
    zone_id UUID REFERENCES facilities.location_zones(zone_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
