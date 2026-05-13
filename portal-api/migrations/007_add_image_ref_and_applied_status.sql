-- Add image_ref to releases table
ALTER TABLE releases ADD COLUMN IF NOT EXISTS image_ref TEXT NOT NULL DEFAULT '';
