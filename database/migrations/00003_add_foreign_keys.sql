-- Foreign keys with explicit ON DELETE behaviour. Until now the schema had none
-- (GORM ran with DisableForeignKeyConstraintWhenMigrating), so orphaned rows may
-- exist; they are cleaned up first so the constraints can be added.
--
-- Soft-deleted parents (deleted_at set) are still present as rows, so they never
-- fire these actions; only hard deletes do.
--
-- Choices:
--   * CASCADE for rows owned by their parent (tags, banners, permissions, aliases,
--     embeddings, join rows, invite codes of a deleted user).
--   * SET NULL for optional references (album group, person tagging, key photo,
--     uploader).
--   * RESTRICT for images.album_id: deleting an album that still has images would
--     leak their objects in storage, so images must be removed first (which
--     ImageRepository.DeleteByAlbum does).
--   * faces.image_path and image_tags.image_path are NO ACTION DEFERRABLE INITIALLY
--     DEFERRED. ImageRepository.Upsert deletes the image row and re-inserts it in
--     one transaction while deliberately keeping tagged faces and manual tags, so
--     the reference must only be checked at commit. CASCADE would destroy the kept
--     rows and an immediate check would make the delete fail. Callers that really
--     delete an image (DeleteImages) already remove faces/embeddings/tags first.
--
-- album_banners.image_path, collection_banners.image_path and
-- album_groups.banner_image_path are storage object keys, not image references, so
-- they have no foreign key.

-- +goose Up

-- Orphan cleanup, children before parents.

-- images without an album cannot be reached; remove them and their dependents.
DELETE FROM face_embeddings WHERE face_id IN (
    SELECT f.id FROM faces f JOIN images i ON i.original_path = f.image_path
    WHERE NOT EXISTS (SELECT 1 FROM albums a WHERE a.id = i.album_id));
DELETE FROM faces WHERE image_path IN (
    SELECT i.original_path FROM images i
    WHERE NOT EXISTS (SELECT 1 FROM albums a WHERE a.id = i.album_id));
DELETE FROM image_tags WHERE image_path IN (
    SELECT i.original_path FROM images i
    WHERE NOT EXISTS (SELECT 1 FROM albums a WHERE a.id = i.album_id));
DELETE FROM images i WHERE NOT EXISTS (SELECT 1 FROM albums a WHERE a.id = i.album_id);

-- faces of images that no longer exist
DELETE FROM face_embeddings WHERE face_id IN (
    SELECT f.id FROM faces f
    WHERE NOT EXISTS (SELECT 1 FROM images i WHERE i.original_path = f.image_path));
DELETE FROM faces f WHERE NOT EXISTS (SELECT 1 FROM images i WHERE i.original_path = f.image_path);
DELETE FROM face_embeddings e WHERE NOT EXISTS (SELECT 1 FROM faces f WHERE f.id = e.face_id);
DELETE FROM image_tags t WHERE NOT EXISTS (SELECT 1 FROM images i WHERE i.original_path = t.image_path);

-- optional references: detach instead of deleting
UPDATE faces f SET person_id = NULL
    WHERE person_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM people p WHERE p.id = f.person_id);
UPDATE people p SET key_photo_face_id = NULL
    WHERE key_photo_face_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM faces f WHERE f.id = p.key_photo_face_id);
UPDATE albums a SET group_id = NULL
    WHERE group_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM album_groups g WHERE g.id = a.group_id);
UPDATE images i SET uploaded_by_user_id = NULL
    WHERE uploaded_by_user_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM users u WHERE u.id = i.uploaded_by_user_id);

-- owned children of missing parents
DELETE FROM aliases x WHERE NOT EXISTS (SELECT 1 FROM people p WHERE p.id = x.person_id);
DELETE FROM album_banners x WHERE NOT EXISTS (SELECT 1 FROM albums a WHERE a.id = x.album_id);
DELETE FROM album_default_tags x WHERE NOT EXISTS (SELECT 1 FROM albums a WHERE a.id = x.album_id);
DELETE FROM user_album_permissions x WHERE
    (x.user_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM users u WHERE u.id = x.user_id))
    OR (x.album_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM albums a WHERE a.id = x.album_id));
DELETE FROM role_album_permissions x WHERE
    (x.role_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM roles r WHERE r.id = x.role_id))
    OR (x.album_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM albums a WHERE a.id = x.album_id));
DELETE FROM user_roles x WHERE
    NOT EXISTS (SELECT 1 FROM users u WHERE u.id = x.user_id)
    OR NOT EXISTS (SELECT 1 FROM roles r WHERE r.id = x.role_id);
DELETE FROM collection_tag_filters x WHERE NOT EXISTS (SELECT 1 FROM collections c WHERE c.id = x.collection_id);
DELETE FROM collection_banners x WHERE NOT EXISTS (SELECT 1 FROM collections c WHERE c.id = x.collection_id);
DELETE FROM invite_codes x WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.id = x.created_by_user_id);

-- images
ALTER TABLE images ADD CONSTRAINT fk_images_album
    FOREIGN KEY (album_id) REFERENCES albums (id) ON DELETE RESTRICT;
ALTER TABLE images ADD CONSTRAINT fk_images_uploaded_by_user
    FOREIGN KEY (uploaded_by_user_id) REFERENCES users (id) ON DELETE SET NULL;

-- albums
ALTER TABLE albums ADD CONSTRAINT fk_albums_group
    FOREIGN KEY (group_id) REFERENCES album_groups (id) ON DELETE SET NULL;
ALTER TABLE album_banners ADD CONSTRAINT fk_album_banners_album
    FOREIGN KEY (album_id) REFERENCES albums (id) ON DELETE CASCADE;
ALTER TABLE album_default_tags ADD CONSTRAINT fk_album_default_tags_album
    FOREIGN KEY (album_id) REFERENCES albums (id) ON DELETE CASCADE;

-- image tags and faces: see the header for why these are deferred
ALTER TABLE image_tags ADD CONSTRAINT fk_image_tags_image
    FOREIGN KEY (image_path) REFERENCES images (original_path)
    DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE faces ADD CONSTRAINT fk_faces_image
    FOREIGN KEY (image_path) REFERENCES images (original_path)
    DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE faces ADD CONSTRAINT fk_faces_person
    FOREIGN KEY (person_id) REFERENCES people (id) ON DELETE SET NULL;
ALTER TABLE face_embeddings ADD CONSTRAINT fk_face_embeddings_face
    FOREIGN KEY (face_id) REFERENCES faces (id) ON DELETE CASCADE;

-- people
ALTER TABLE people ADD CONSTRAINT fk_people_key_photo_face
    FOREIGN KEY (key_photo_face_id) REFERENCES faces (id) ON DELETE SET NULL;
ALTER TABLE aliases ADD CONSTRAINT fk_aliases_person
    FOREIGN KEY (person_id) REFERENCES people (id) ON DELETE CASCADE;

-- collections
ALTER TABLE collection_tag_filters ADD CONSTRAINT fk_collection_tag_filters_collection
    FOREIGN KEY (collection_id) REFERENCES collections (id) ON DELETE CASCADE;
ALTER TABLE collection_banners ADD CONSTRAINT fk_collection_banners_collection
    FOREIGN KEY (collection_id) REFERENCES collections (id) ON DELETE CASCADE;

-- users, roles and permissions
ALTER TABLE user_roles ADD CONSTRAINT fk_user_roles_user
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE user_roles ADD CONSTRAINT fk_user_roles_role
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE;
ALTER TABLE user_album_permissions ADD CONSTRAINT fk_user_album_permissions_user
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE user_album_permissions ADD CONSTRAINT fk_user_album_permissions_album
    FOREIGN KEY (album_id) REFERENCES albums (id) ON DELETE CASCADE;
ALTER TABLE role_album_permissions ADD CONSTRAINT fk_role_album_permissions_role
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE;
ALTER TABLE role_album_permissions ADD CONSTRAINT fk_role_album_permissions_album
    FOREIGN KEY (album_id) REFERENCES albums (id) ON DELETE CASCADE;
-- created_by_user_id is NOT NULL (and a plain uint in the model), so SET NULL is not
-- possible; invite codes go away with the user who created them.
ALTER TABLE invite_codes ADD CONSTRAINT fk_invite_codes_created_by_user
    FOREIGN KEY (created_by_user_id) REFERENCES users (id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE invite_codes DROP CONSTRAINT IF EXISTS fk_invite_codes_created_by_user;
ALTER TABLE role_album_permissions DROP CONSTRAINT IF EXISTS fk_role_album_permissions_album;
ALTER TABLE role_album_permissions DROP CONSTRAINT IF EXISTS fk_role_album_permissions_role;
ALTER TABLE user_album_permissions DROP CONSTRAINT IF EXISTS fk_user_album_permissions_album;
ALTER TABLE user_album_permissions DROP CONSTRAINT IF EXISTS fk_user_album_permissions_user;
ALTER TABLE user_roles DROP CONSTRAINT IF EXISTS fk_user_roles_role;
ALTER TABLE user_roles DROP CONSTRAINT IF EXISTS fk_user_roles_user;
ALTER TABLE collection_banners DROP CONSTRAINT IF EXISTS fk_collection_banners_collection;
ALTER TABLE collection_tag_filters DROP CONSTRAINT IF EXISTS fk_collection_tag_filters_collection;
ALTER TABLE aliases DROP CONSTRAINT IF EXISTS fk_aliases_person;
ALTER TABLE people DROP CONSTRAINT IF EXISTS fk_people_key_photo_face;
ALTER TABLE face_embeddings DROP CONSTRAINT IF EXISTS fk_face_embeddings_face;
ALTER TABLE faces DROP CONSTRAINT IF EXISTS fk_faces_person;
ALTER TABLE faces DROP CONSTRAINT IF EXISTS fk_faces_image;
ALTER TABLE image_tags DROP CONSTRAINT IF EXISTS fk_image_tags_image;
ALTER TABLE album_default_tags DROP CONSTRAINT IF EXISTS fk_album_default_tags_album;
ALTER TABLE album_banners DROP CONSTRAINT IF EXISTS fk_album_banners_album;
ALTER TABLE albums DROP CONSTRAINT IF EXISTS fk_albums_group;
ALTER TABLE images DROP CONSTRAINT IF EXISTS fk_images_uploaded_by_user;
ALTER TABLE images DROP CONSTRAINT IF EXISTS fk_images_album;
