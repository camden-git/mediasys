package database_test

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/camden-git/mediasysbackend/database"
)

// TestForeignKeyDeleteBehavior checks the ON DELETE behaviour chosen in
// 00003_add_foreign_keys.sql, in particular that an image row can be deleted and
// re-inserted in one transaction (what ImageRepository.Upsert does) without losing
// its tagged faces or manual tags. Needs TEST_DATABASE_URL like the other schema tests.
func TestForeignKeyDeleteBehavior(t *testing.T) {
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		skipOrFailMissingEnv(t, "TEST_DATABASE_URL not set; skipping foreign key test")
	}
	ctx := context.Background()

	adminDB, err := sql.Open("pgx", adminURL)
	if err != nil {
		t.Fatalf("failed to open admin connection: %v", err)
	}
	defer adminDB.Close()

	dbName := fmt.Sprintf("mediasys_fk_test_%d", rand.Intn(1_000_000))
	createDatabase(t, adminDB, dbName)
	defer dropDatabase(t, adminDB, dbName)

	db, err := sql.Open("pgx", withDBName(t, adminURL, dbName))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	if err := database.RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	count := func(q string) int {
		t.Helper()
		var n int
		if err := db.QueryRow(q).Scan(&n); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return n
	}
	const insertImage = `INSERT INTO images (original_path, album_id, object_key, last_modified, created_at)
		VALUES ('a/x.jpg', 1, 'a/x.jpg', 1, 1)`

	mustExec(`INSERT INTO albums (id, name, slug, folder_path, created_at, updated_at) VALUES (1, 'a', 'a', 'a', 1, 1)`)
	mustExec(`INSERT INTO people (id, primary_name, created_at, updated_at) VALUES (1, 'p', 1, 1)`)
	mustExec(insertImage)
	mustExec(`INSERT INTO faces (id, person_id, image_path, x1, y1, x2, y2, created_at, updated_at)
		VALUES (1, 1, 'a/x.jpg', 0, 0, 1, 1, 1, 1)`)
	mustExec(`INSERT INTO face_embeddings (face_id, embedding, created_at, updated_at)
		VALUES (1, array_fill(0, ARRAY[512])::vector, 1, 1)`)
	mustExec(`UPDATE people SET key_photo_face_id = 1 WHERE id = 1`)
	mustExec(`INSERT INTO aliases (person_id, name) VALUES (1, 'alias')`)
	mustExec(`INSERT INTO image_tags (image_path, tag_key, tag_value, source, created_at)
		VALUES ('a/x.jpg', 'k', 'v', 'manual', now())`)

	// Upsert-style delete + re-insert in one transaction keeps tagged faces and tags.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`DELETE FROM images WHERE original_path = 'a/x.jpg'`); err != nil {
		t.Fatalf("deleting image inside transaction failed: %v", err)
	}
	if _, err := tx.Exec(insertImage); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit of delete+reinsert failed: %v", err)
	}
	if n := count(`SELECT count(*) FROM faces`) + count(`SELECT count(*) FROM image_tags`); n != 2 {
		t.Errorf("tagged face / manual tag lost on delete+reinsert (rows=%d)", n)
	}

	// Really deleting an image that still has faces or tags must fail.
	if _, err := db.Exec(`DELETE FROM images WHERE original_path = 'a/x.jpg'`); err == nil {
		t.Errorf("deleting an image with dependent faces/tags should fail")
	}

	// Deleting a person untags faces and removes aliases.
	mustExec(`DELETE FROM people WHERE id = 1`)
	if n := count(`SELECT count(*) FROM faces WHERE person_id IS NULL`); n != 1 {
		t.Errorf("face was not untagged when person deleted")
	}
	if n := count(`SELECT count(*) FROM aliases`); n != 0 {
		t.Errorf("aliases not removed with person")
	}

	// Deleting a face removes its embedding.
	mustExec(`DELETE FROM faces WHERE id = 1`)
	if n := count(`SELECT count(*) FROM face_embeddings`); n != 0 {
		t.Errorf("embedding not removed with face")
	}

	// Albums with images cannot be hard-deleted.
	if _, err := db.Exec(`DELETE FROM albums WHERE id = 1`); err == nil {
		t.Errorf("deleting an album that still has images should fail")
	}
}
