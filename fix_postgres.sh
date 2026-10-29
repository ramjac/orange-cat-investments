#!/bin/bash

# Fix init.sql (reverting exporter role)
sed -i '/-- Create exporter role for monitoring/,/GRANT pg_monitor TO postgres_exporter;/d' scripts/init.sql
sed -i '/-- Create exporter role for monitoring/,/GRANT pg_monitor TO postgres_exporter;/d' k8s/oci-core/postgres-core-configmap.yaml
sed -i '/-- Create exporter role for monitoring/,/GRANT pg_monitor TO postgres_exporter;/d' k8s/oci-dev/postgres-dev-configmap.yaml
sed -i '/-- Create exporter role for monitoring/,/GRANT pg_monitor TO postgres_exporter;/d' k8s/oci-internal/postgres-internal.yaml
