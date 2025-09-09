-- Add image metadata fields to events table
ALTER TABLE events 
ADD COLUMN image_key VARCHAR(255),
ADD COLUMN image_size BIGINT DEFAULT 0,
ADD COLUMN image_format VARCHAR(20),
ADD COLUMN image_width INTEGER DEFAULT 0,
ADD COLUMN image_height INTEGER DEFAULT 0,
ADD COLUMN image_uploaded_at TIMESTAMP;

-- Add indexes for image-related queries
CREATE INDEX idx_events_image_key ON events(image_key) WHERE image_key IS NOT NULL;
CREATE INDEX idx_events_has_image ON events(id) WHERE image_url IS NOT NULL AND image_key IS NOT NULL;

-- Add constraints for image metadata consistency
ALTER TABLE events ADD CONSTRAINT chk_image_metadata_consistency 
CHECK (
    (image_url IS NULL AND image_key IS NULL AND image_size = 0 AND image_format IS NULL AND image_width = 0 AND image_height = 0 AND image_uploaded_at IS NULL) OR
    (image_url IS NOT NULL AND image_key IS NOT NULL AND image_size > 0 AND image_format IS NOT NULL AND image_width > 0 AND image_height > 0)
);

-- Add constraint for valid image formats
ALTER TABLE events ADD CONSTRAINT chk_image_format 
CHECK (image_format IS NULL OR image_format IN ('jpeg', 'jpg', 'png', 'webp', 'gif'));

-- Add constraint for reasonable image dimensions
ALTER TABLE events ADD CONSTRAINT chk_image_dimensions 
CHECK (
    (image_width = 0 AND image_height = 0) OR 
    (image_width > 0 AND image_height > 0 AND image_width <= 10000 AND image_height <= 10000)
);

-- Add constraint for reasonable image size (max 10MB)
ALTER TABLE events ADD CONSTRAINT chk_image_size 
CHECK (image_size >= 0 AND image_size <= 10485760);