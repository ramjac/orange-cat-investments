-- OCI Database Bootstrap Script: Extensions and RFC 4122 UUIDv7 generator
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Function to generate RFC 4122 compliant UUIDv7 in PostgreSQL 16
CREATE OR REPLACE FUNCTION gen_random_uuid_v7()
RETURNS uuid AS $$
DECLARE
    v_time timestamp with time zone := clock_timestamp();
    v_secs bigint := extract(epoch from v_time);
    v_msec bigint := (extract(milliseconds from v_time)::bigint) % 1000;
    v_timestamp bigint := (v_secs * 1000) + v_msec;
    v_timestamp_hex text := lpad(to_hex(v_timestamp), 12, '0');
    v_bytes bytea := decode(v_timestamp_hex || encode(gen_random_bytes(10), 'hex'), 'hex');
BEGIN
    -- Set version to 7 (bits 48-51 to 0111)
    v_bytes := set_byte(v_bytes, 6, (get_byte(v_bytes, 6) & 15) | 112);
    -- Set variant to RFC 4122 (bits 64-65 to 10)
    v_bytes := set_byte(v_bytes, 8, (get_byte(v_bytes, 8) & 63) | 128);
    RETURN encode(v_bytes, 'hex')::uuid;
END;
$$ LANGUAGE plpgsql VOLATILE;
