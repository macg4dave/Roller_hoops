-- +migrate Up
+
+CREATE EXTENSION IF NOT EXISTS pgcrypto;
+
+CREATE TABLE IF NOT EXISTS devices (
+  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
+  display_name text NULL,
+  created_at timestamptz NOT NULL DEFAULT now(),
+  updated_at timestamptz NOT NULL DEFAULT now()
+);
+
+CREATE TABLE IF NOT EXISTS interfaces (
+  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
+  device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
+  created_at timestamptz NOT NULL DEFAULT now(),
+  updated_at timestamptz NOT NULL DEFAULT now()
+);
+
+CREATE TABLE IF NOT EXISTS ip_addresses (
+  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
+  device_id uuid NULL REFERENCES devices(id) ON DELETE CASCADE,
+  interface_id uuid NULL REFERENCES interfaces(id) ON DELETE CASCADE,
+  ip inet NOT NULL,
+  created_at timestamptz NOT NULL DEFAULT now(),
+  updated_at timestamptz NOT NULL DEFAULT now()
+);
+
+CREATE TABLE IF NOT EXISTS mac_addresses (
+  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
+  device_id uuid NULL REFERENCES devices(id) ON DELETE CASCADE,
+  interface_id uuid NULL REFERENCES interfaces(id) ON DELETE CASCADE,
+  mac macaddr NOT NULL,
+  created_at timestamptz NOT NULL DEFAULT now(),
+  updated_at timestamptz NOT NULL DEFAULT now()
+);
+
+CREATE TABLE IF NOT EXISTS services (
+  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
+  device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
+  created_at timestamptz NOT NULL DEFAULT now(),
+  updated_at timestamptz NOT NULL DEFAULT now()
+);
+
+CREATE TABLE IF NOT EXISTS device_metadata (
+  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
+  device_id uuid NOT NULL UNIQUE REFERENCES devices(id) ON DELETE CASCADE,
+  created_at timestamptz NOT NULL DEFAULT now(),
+  updated_at timestamptz NOT NULL DEFAULT now()
+);
