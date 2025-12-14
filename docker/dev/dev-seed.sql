-- Development seed data
-- The script is idempotent so it can be rerun via the dev profile.
-- Used by: `docker compose --profile dev up`

DO $$
DECLARE
  dev_id uuid;
  iface_id uuid;
BEGIN
  SELECT id INTO dev_id FROM devices WHERE display_name = 'Seeded Office Router';
  IF dev_id IS NULL THEN
    INSERT INTO devices (display_name)
    VALUES ('Seeded Office Router')
    RETURNING id INTO dev_id;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM device_metadata WHERE device_id = dev_id) THEN
    INSERT INTO device_metadata (device_id, owner, location, notes)
    VALUES (dev_id, 'Infrastructure Team', 'Data Center Lab', 'Seeded device for dev/staging use');
  END IF;

  SELECT id INTO iface_id FROM interfaces WHERE device_id = dev_id AND name = 'eth0';
  IF iface_id IS NULL THEN
    INSERT INTO interfaces (device_id, name, ifindex)
    VALUES (dev_id, 'eth0', 1)
    RETURNING id INTO iface_id;
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM ip_addresses
    WHERE interface_id = iface_id
      AND ip = '192.168.100.10'
  ) THEN
    INSERT INTO ip_addresses (device_id, interface_id, ip)
    VALUES (dev_id, iface_id, '192.168.100.10');
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM mac_addresses
    WHERE interface_id = iface_id
      AND mac = 'aa:bb:cc:dd:ee:ff'
  ) THEN
    INSERT INTO mac_addresses (device_id, interface_id, mac)
    VALUES (dev_id, iface_id, 'aa:bb:cc:dd:ee:ff');
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM services
    WHERE device_id = dev_id
      AND name = 'ssh'
  ) THEN
    INSERT INTO services (device_id, protocol, port, name)
    VALUES (dev_id, 'tcp', 22, 'ssh');
  END IF;
END $$;
