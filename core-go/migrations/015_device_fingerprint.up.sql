-- Add OS guess and MAC vendor fields to devices table
ALTER TABLE devices ADD COLUMN IF NOT EXISTS os_guess text;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS os_guess_confidence text;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS mac_vendor text;
