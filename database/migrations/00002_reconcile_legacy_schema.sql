-- Reconciles databases that were created by GORM AutoMigrate before goose was
-- introduced (baselined at version 1 by database.RunMigrations) with the schema
-- defined by 00001_initial_schema.sql. Every statement is idempotent, so this is a
-- no-op on databases that were created by 00001 itself.

-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS vector;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE COLLATION IF NOT EXISTS natural_sort (provider = icu, locale = 'und-u-kn-true');
-- +goose StatementEnd

-- Before c169d4a name/slug/folder_path used a plain `unique` tag, which GORM created
-- as table-wide constraints (uni_<table>_<column>). Those block re-using a name or
-- slug after a soft delete, so drop them in favour of the partial unique indexes
-- created below.
ALTER TABLE album_groups DROP CONSTRAINT IF EXISTS uni_album_groups_name;
ALTER TABLE album_groups DROP CONSTRAINT IF EXISTS uni_album_groups_slug;
ALTER TABLE albums DROP CONSTRAINT IF EXISTS uni_albums_name;
ALTER TABLE albums DROP CONSTRAINT IF EXISTS uni_albums_slug;
ALTER TABLE albums DROP CONSTRAINT IF EXISTS uni_albums_folder_path;
ALTER TABLE collections DROP CONSTRAINT IF EXISTS uni_collections_name;
ALTER TABLE collections DROP CONSTRAINT IF EXISTS uni_collections_slug;
-- ...or as bare unique indexes, depending on the GORM version.
DROP INDEX IF EXISTS uni_album_groups_name;
DROP INDEX IF EXISTS uni_album_groups_slug;
DROP INDEX IF EXISTS uni_albums_name;
DROP INDEX IF EXISTS uni_albums_slug;
DROP INDEX IF EXISTS uni_albums_folder_path;
DROP INDEX IF EXISTS uni_collections_name;
DROP INDEX IF EXISTS uni_collections_slug;

-- Every index from 00001, created only where missing. This covers the partial unique
-- indexes (c169d4a), the album listing composite indexes (b3add31) and the pgvector
-- HNSW index (b712726), which older AutoMigrate databases may not have.
CREATE UNIQUE INDEX IF NOT EXISTS idx_album_groups_name ON album_groups (name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_album_groups_slug ON album_groups (slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_album_groups_deleted_at ON album_groups (deleted_at);
CREATE INDEX IF NOT EXISTS idx_people_key_photo_face_id ON people (key_photo_face_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_person_name ON aliases (person_id, name);
CREATE INDEX IF NOT EXISTS idx_faces_person_id ON faces (person_id);
CREATE INDEX IF NOT EXISTS idx_faces_image_path ON faces (image_path);
CREATE INDEX IF NOT EXISTS idx_faces_deleted_at ON faces (deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_face_embeddings_face_id ON face_embeddings (face_id);
CREATE INDEX IF NOT EXISTS idx_face_embeddings_deleted_at ON face_embeddings (deleted_at);
CREATE INDEX IF NOT EXISTS idx_face_embeddings_embedding_hnsw_cosine ON face_embeddings USING hnsw (embedding vector_cosine_ops);
CREATE UNIQUE INDEX IF NOT EXISTS idx_albums_name ON albums (name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_albums_slug ON albums (slug) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_albums_folder_path ON albums (folder_path) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_albums_group_id ON albums (group_id);
CREATE INDEX IF NOT EXISTS idx_albums_deleted_at ON albums (deleted_at);
CREATE INDEX IF NOT EXISTS idx_album_banners_album_id ON album_banners (album_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_album_default_tags_unique ON album_default_tags (album_id, tag_key, tag_value);
CREATE UNIQUE INDEX IF NOT EXISTS idx_images_object_key ON images (object_key);
CREATE INDEX IF NOT EXISTS idx_images_uploaded_by_user_id ON images (uploaded_by_user_id);
CREATE INDEX IF NOT EXISTS idx_images_deleted_at ON images (deleted_at);
CREATE INDEX IF NOT EXISTS idx_images_album_taken ON images (album_id, taken_at);
CREATE INDEX IF NOT EXISTS idx_images_album_modified ON images (album_id, last_modified);
CREATE INDEX IF NOT EXISTS idx_image_tags_image_path ON image_tags (image_path);
CREATE UNIQUE INDEX IF NOT EXISTS idx_image_tags_unique ON image_tags (image_path, tag_key, tag_value, source);
CREATE INDEX IF NOT EXISTS idx_image_tags_key_value ON image_tags (tag_key, tag_value);
CREATE UNIQUE INDEX IF NOT EXISTS idx_collections_name ON collections (name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_collections_slug ON collections (slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_collections_deleted_at ON collections (deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_collection_tag_filters_unique ON collection_tag_filters (collection_id, tag_key, tag_value);
CREATE INDEX IF NOT EXISTS idx_collection_banners_collection_id ON collection_banners (collection_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_album ON user_album_permissions (user_id, album_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_name ON roles (name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_role_album ON role_album_permissions (role_id, album_id);
CREATE INDEX IF NOT EXISTS idx_invite_codes_created_by_user_id ON invite_codes (created_by_user_id);

-- +goose Down
-- Reconciliation of legacy schemas is not reversible; nothing to undo.
SELECT 1;
