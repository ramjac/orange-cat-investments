DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'oci_monitor') THEN
    CREATE ROLE oci_monitor WITH LOGIN PASSWORD 'oci_secure_pass';
  ELSE
    ALTER ROLE oci_monitor WITH PASSWORD 'oci_secure_pass';
  END IF;
END
$$;
GRANT pg_monitor TO oci_monitor;
