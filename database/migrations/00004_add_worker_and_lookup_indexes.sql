-- +goose Up

-- Image worker dispatcher (ImageRepository.ListPendingProcessing /
-- ListPendingDetection, polled every 15s): find the oldest images with pending
-- work. Partial indexes stay tiny because almost every image is done.
CREATE INDEX IF NOT EXISTS idx_images_pending_processing ON images (created_at)
    WHERE deleted_at IS NULL
      AND (metadata_status = 'pending' OR thumbnail_status = 'pending' OR preview_status = 'pending');
CREATE INDEX IF NOT EXISTS idx_images_pending_detection ON images (created_at)
    WHERE deleted_at IS NULL AND detection_status = 'pending';

-- Lookups by the second half of the join/permission tables (the unique indexes
-- lead with user_id / role_id), and cheap cascades for the foreign keys.
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles (role_id);
CREATE INDEX IF NOT EXISTS idx_user_album_permissions_album_id ON user_album_permissions (album_id);
CREATE INDEX IF NOT EXISTS idx_role_album_permissions_album_id ON role_album_permissions (album_id);

-- +goose Down
DROP INDEX IF EXISTS idx_role_album_permissions_album_id;
DROP INDEX IF EXISTS idx_user_album_permissions_album_id;
DROP INDEX IF EXISTS idx_user_roles_role_id;
DROP INDEX IF EXISTS idx_images_pending_detection;
DROP INDEX IF EXISTS idx_images_pending_processing;
