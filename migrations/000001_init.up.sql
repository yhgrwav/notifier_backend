CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS incidents (
                                         id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title         VARCHAR(255) NOT NULL,
    description   TEXT,
    lat     DOUBLE PRECISION NOT NULL,
    lon     DOUBLE PRECISION NOT NULL,
    radius_meters DOUBLE PRECISION NOT NULL,
    is_active     BOOLEAN DEFAULT TRUE,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
                                );

CREATE TABLE IF NOT EXISTS location_checks (
                                               id          BIGSERIAL PRIMARY KEY,
                                               user_id     VARCHAR(255) NOT NULL,
    lat         DOUBLE PRECISION NOT NULL,
    lon         DOUBLE PRECISION NOT NULL,
    incident_ids UUID[] DEFAULT '{}',
    checked_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
                              );

CREATE INDEX IF NOT EXISTS idx_location_checks_incident_ids ON location_checks USING GIN (incident_ids);
CREATE INDEX IF NOT EXISTS idx_location_checks_time ON location_checks(checked_at);