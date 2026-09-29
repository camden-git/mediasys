-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS vector;
-- +goose StatementEnd

-- ICU collation used by database.SQLOrderClause for the "filename_nat" sort order
-- (natural sort, e.g. "img2" before "img10"). Requires a Postgres build with ICU
-- support, which is standard on the official postgres image (and images built on
-- top of it, such as pgvector/pgvector) since Postgres 15.
-- +goose StatementBegin
CREATE COLLATION IF NOT EXISTS natural_sort (provider = icu, locale = 'und-u-kn-true');
-- +goose StatementEnd

-- album_groups

CREATE TABLE album_groups (
    id                 bigserial PRIMARY KEY,
    name               text NOT NULL,
    slug               text NOT NULL,
    description        text,
    banner_image_path  text,
    is_hidden          boolean NOT NULL DEFAULT false,
    created_at         bigint NOT NULL,
    updated_at         bigint NOT NULL,
    deleted_at         timestamptz
);

CREATE UNIQUE INDEX idx_album_groups_name ON album_groups (name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_album_groups_slug ON album_groups (slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_album_groups_deleted_at ON album_groups (deleted_at);

-- people

CREATE TABLE people (
    id                 bigserial PRIMARY KEY,
    primary_name       text NOT NULL,
    created_at         bigint NOT NULL,
    updated_at         bigint NOT NULL,
    key_photo_face_id  bigint
);

CREATE INDEX idx_people_key_photo_face_id ON people (key_photo_face_id);

-- aliases

CREATE TABLE aliases (
    id         bigserial PRIMARY KEY,
    person_id  bigint NOT NULL,
    name       text NOT NULL
);

CREATE UNIQUE INDEX idx_person_name ON aliases (person_id, name);

-- faces

CREATE TABLE faces (
    id                       bigserial PRIMARY KEY,
    person_id                bigint,
    confirmed                boolean NOT NULL DEFAULT false,
    image_path               text NOT NULL,
    x1                       bigint NOT NULL,
    y1                       bigint NOT NULL,
    x2                       bigint NOT NULL,
    y2                       bigint NOT NULL,
    detection_confidence     decimal NOT NULL DEFAULT 0,
    recognition_confidence   decimal,
    quality_score            decimal,
    landmarks                text,
    pose_yaw                 decimal,
    pose_pitch               decimal,
    pose_roll                decimal,
    created_at               bigint NOT NULL,
    updated_at               bigint NOT NULL,
    deleted_at               timestamptz
);

CREATE INDEX idx_faces_person_id ON faces (person_id);
CREATE INDEX idx_faces_image_path ON faces (image_path);
CREATE INDEX idx_faces_deleted_at ON faces (deleted_at);

-- face_embeddings

CREATE TABLE face_embeddings (
    id                bigserial PRIMARY KEY,
    face_id           bigint NOT NULL,
    embedding         vector(512) NOT NULL,
    embedding_model   text NOT NULL DEFAULT 'arcface',
    quality_score     decimal,
    created_at        bigint NOT NULL,
    updated_at        bigint NOT NULL,
    deleted_at        timestamptz
);

CREATE UNIQUE INDEX idx_face_embeddings_face_id ON face_embeddings (face_id);
CREATE INDEX idx_face_embeddings_deleted_at ON face_embeddings (deleted_at);

-- HNSW index for approximate nearest-neighbour cosine similarity search over face
-- embeddings. Faces are compared by cosine similarity (see media.FaceRecognitionModel
-- .CalculateSimilarity and handlers.FaceRecognitionService.CalculateSimilarity), so use
-- vector_cosine_ops to match that semantics.
CREATE INDEX idx_face_embeddings_embedding_hnsw_cosine ON face_embeddings USING hnsw (embedding vector_cosine_ops);

-- albums

CREATE TABLE albums (
    id                     bigserial PRIMARY KEY,
    name                   text NOT NULL,
    slug                   text NOT NULL,
    description            text,
    folder_path            text NOT NULL,
    sort_order             text NOT NULL DEFAULT 'filename_asc',
    zip_path               text,
    zip_size               bigint,
    zip_status             text NOT NULL DEFAULT 'notRequired',
    zip_last_generated_at  bigint,
    zip_last_requested_at  bigint,
    zip_error              text,
    created_at             bigint NOT NULL,
    updated_at             bigint NOT NULL,
    is_hidden              boolean NOT NULL DEFAULT false,
    location               text,
    group_id               bigint,
    deleted_at             timestamptz
);

CREATE UNIQUE INDEX idx_albums_name ON albums (name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_albums_slug ON albums (slug) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_albums_folder_path ON albums (folder_path) WHERE deleted_at IS NULL;
CREATE INDEX idx_albums_group_id ON albums (group_id);
CREATE INDEX idx_albums_deleted_at ON albums (deleted_at);

-- album_banners

CREATE TABLE album_banners (
    id          bigserial PRIMARY KEY,
    album_id    bigint NOT NULL,
    image_path  text NOT NULL,
    sort_order  bigint NOT NULL DEFAULT 0,
    created_at  bigint NOT NULL
);

CREATE INDEX idx_album_banners_album_id ON album_banners (album_id);

-- album_default_tags

CREATE TABLE album_default_tags (
    id          bigserial PRIMARY KEY,
    album_id    bigint NOT NULL,
    tag_key     text NOT NULL,
    tag_value   text NOT NULL,
    created_at  timestamptz NOT NULL
);

CREATE UNIQUE INDEX idx_album_default_tags_unique ON album_default_tags (album_id, tag_key, tag_value);

-- images

CREATE TABLE images (
    original_path            text PRIMARY KEY,
    album_id                 bigint NOT NULL,
    object_key                text NOT NULL,
    size                      bigint NOT NULL DEFAULT 0,
    content_type              text NOT NULL DEFAULT '',
    last_modified             bigint NOT NULL,
    created_at                bigint NOT NULL,
    uploaded_by_user_id       bigint,
    width                     bigint,
    height                    bigint,
    taken_at                  bigint,
    camera_make               text,
    camera_model              text,
    lens_make                 text,
    lens_model                text,
    focal_length              decimal,
    aperture                  decimal,
    shutter_speed             text,
    iso                       bigint,
    rating                    bigint,
    thumbnail_path            text,
    preview_path              text,
    metadata_status           text NOT NULL DEFAULT 'pending',
    thumbnail_status          text NOT NULL DEFAULT 'pending',
    preview_status            text NOT NULL DEFAULT 'pending',
    detection_status          text NOT NULL DEFAULT 'pending',
    metadata_processed_at     bigint,
    thumbnail_processed_at    bigint,
    preview_processed_at      bigint,
    detection_processed_at    bigint,
    metadata_error            text,
    thumbnail_error           text,
    preview_error             text,
    detection_error           text,
    deleted_at                timestamptz
);

CREATE UNIQUE INDEX idx_images_object_key ON images (object_key);
CREATE INDEX idx_images_uploaded_by_user_id ON images (uploaded_by_user_id);
CREATE INDEX idx_images_deleted_at ON images (deleted_at);
-- composite indexes supporting "images in an album, ordered by capture/mod time"
-- listing queries; see database.SQLOrderClause.
CREATE INDEX idx_images_album_taken ON images (album_id, taken_at);
CREATE INDEX idx_images_album_modified ON images (album_id, last_modified);

-- image_tags

CREATE TABLE image_tags (
    id          bigserial PRIMARY KEY,
    image_path  text NOT NULL,
    tag_key     text NOT NULL,
    tag_value   text NOT NULL,
    source      text NOT NULL,
    created_at  timestamptz NOT NULL
);

CREATE INDEX idx_image_tags_image_path ON image_tags (image_path);
CREATE UNIQUE INDEX idx_image_tags_unique ON image_tags (image_path, tag_key, tag_value, source);
CREATE INDEX idx_image_tags_key_value ON image_tags (tag_key, tag_value);

-- collections

CREATE TABLE collections (
    id                            bigserial PRIMARY KEY,
    name                          text NOT NULL,
    slug                          text NOT NULL,
    description                   text,
    inherit_banners_from_albums   boolean NOT NULL DEFAULT false,
    is_public                     boolean NOT NULL,
    filter_match                  text NOT NULL DEFAULT 'all',
    sort_order                    text NOT NULL DEFAULT 'filename_asc',
    created_at                    bigint NOT NULL,
    updated_at                    bigint NOT NULL,
    deleted_at                    timestamptz
);

CREATE UNIQUE INDEX idx_collections_name ON collections (name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_collections_slug ON collections (slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_collections_deleted_at ON collections (deleted_at);

-- collection_tag_filters

CREATE TABLE collection_tag_filters (
    id             bigserial PRIMARY KEY,
    collection_id  bigint NOT NULL,
    tag_key        text NOT NULL,
    tag_value      text NOT NULL,
    negate         boolean NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX idx_collection_tag_filters_unique ON collection_tag_filters (collection_id, tag_key, tag_value);

-- collection_banners

CREATE TABLE collection_banners (
    id             bigserial PRIMARY KEY,
    collection_id  bigint NOT NULL,
    image_path     text NOT NULL,
    sort_order     bigint NOT NULL DEFAULT 0,
    created_at     bigint NOT NULL
);

CREATE INDEX idx_collection_banners_collection_id ON collection_banners (collection_id);

-- users

CREATE TABLE users (
    id                  bigserial PRIMARY KEY,
    username            text NOT NULL,
    first_name          text,
    last_name           text,
    password_hash       text NOT NULL,
    global_permissions  text,
    created_at          timestamptz,
    updated_at          timestamptz
);

CREATE UNIQUE INDEX idx_users_username ON users (username);

-- user_album_permissions

CREATE TABLE user_album_permissions (
    id           bigserial PRIMARY KEY,
    user_id      bigint,
    album_id     bigint,
    permissions  text,
    created_at   timestamptz,
    updated_at   timestamptz
);

CREATE UNIQUE INDEX idx_user_album ON user_album_permissions (user_id, album_id);

-- roles

CREATE TABLE roles (
    id                        bigserial PRIMARY KEY,
    name                      text NOT NULL,
    global_permissions        text,
    global_album_permissions  text,
    created_at                timestamptz,
    updated_at                timestamptz
);

CREATE UNIQUE INDEX idx_roles_name ON roles (name);

-- user_roles (join table for the User<->Role many2many relationship; also modelled
-- explicitly as models.UserRole so it can carry created_at/updated_at)

CREATE TABLE user_roles (
    user_id     bigint NOT NULL,
    role_id     bigint NOT NULL,
    created_at  timestamptz,
    updated_at  timestamptz,
    PRIMARY KEY (user_id, role_id)
);

-- role_album_permissions

CREATE TABLE role_album_permissions (
    id           bigserial PRIMARY KEY,
    role_id      bigint,
    album_id     bigint,
    permissions  text,
    created_at   timestamptz,
    updated_at   timestamptz
);

CREATE UNIQUE INDEX idx_role_album ON role_album_permissions (role_id, album_id);

-- invite_codes

CREATE TABLE invite_codes (
    id                   bigserial PRIMARY KEY,
    code                 varchar(32) NOT NULL,
    expires_at           timestamptz,
    max_uses             bigint,
    uses                 bigint,
    is_active            boolean NOT NULL DEFAULT true,
    created_by_user_id   bigint NOT NULL,
    created_at           timestamptz,
    updated_at           timestamptz,
    CONSTRAINT uni_invite_codes_code UNIQUE (code)
);

CREATE INDEX idx_invite_codes_created_by_user_id ON invite_codes (created_by_user_id);

-- +goose Down
DROP TABLE IF EXISTS invite_codes;
DROP TABLE IF EXISTS role_album_permissions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS user_album_permissions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS collection_banners;
DROP TABLE IF EXISTS collection_tag_filters;
DROP TABLE IF EXISTS collections;
DROP TABLE IF EXISTS image_tags;
DROP TABLE IF EXISTS images;
DROP TABLE IF EXISTS album_default_tags;
DROP TABLE IF EXISTS album_banners;
DROP TABLE IF EXISTS albums;
DROP TABLE IF EXISTS face_embeddings;
DROP TABLE IF EXISTS faces;
DROP TABLE IF EXISTS aliases;
DROP TABLE IF EXISTS people;
DROP TABLE IF EXISTS album_groups;
DROP COLLATION IF EXISTS natural_sort;
DROP EXTENSION IF EXISTS vector;
