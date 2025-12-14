-- +migrate Down

DROP TABLE IF EXISTS device_metadata;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS mac_addresses;
DROP TABLE IF EXISTS ip_addresses;
DROP TABLE IF EXISTS interfaces;
DROP TABLE IF EXISTS devices;

DROP EXTENSION IF EXISTS pgcrypto;
