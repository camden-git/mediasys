package database

import (
	"database/sql"
	"fmt"
	"github.com/camden-git/mediasysbackend/media"
	"log"
	"path/filepath"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type Image struct {
	OriginalPath  string
	ThumbnailPath *string
	LastModified  int64
	Width         *int
	Height        *int
	Aperture      *float64
	ShutterSpeed  *string
	ISO           *int
	FocalLength   *float64
	LensMake      *string
	LensModel     *string
	CameraMake    *string
	CameraModel   *string
	TakenAt       *int64

	// statuses
	ThumbnailStatus string
	MetadataStatus  string
	DetectionStatus string

	// timestamps
	ThumbnailProcessedAt *int64
	MetadataProcessedAt  *int64
	DetectionProcessedAt *int64

	// errors
	ThumbnailError *string
	MetadataError  *string
	DetectionError *string
}

// GetImageInfo retrieves full image info
func GetImageInfo(db Querier, originalPath string) (Image, error) {
	var info Image
	queryBuilder := psql.Select(
		"original_path", "thumbnail_path", "last_modified", "width", "height",
		"aperture", "shutter_speed", "iso", "focal_length",
		"lens_make", "lens_model", "camera_make", "camera_model", "taken_at",

		"thumbnail_status", "metadata_status", "detection_status", // statuses
		"thumbnail_processed_at", "metadata_processed_at", "detection_processed_at", // timestamps
		"thumbnail_error", "metadata_error", "detection_error", // errors
	).From("images").
		Where(sq.Eq{"original_path": originalPath}).
		Limit(1)

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return Image{}, fmt.Errorf("failed to build SQL query for GetImageInfo: %w", err)
	}

	err = db.QueryRow(sqlStr, args...).Scan(
		&info.OriginalPath, &info.ThumbnailPath, &info.LastModified, &info.Width, &info.Height,
		&info.Aperture, &info.ShutterSpeed, &info.ISO, &info.FocalLength,
		&info.LensMake, &info.LensModel, &info.CameraMake, &info.CameraModel, &info.TakenAt,
		&info.ThumbnailStatus, &info.MetadataStatus, &info.DetectionStatus,
		&info.ThumbnailProcessedAt, &info.MetadataProcessedAt, &info.DetectionProcessedAt,
		&info.ThumbnailError, &info.MetadataError, &info.DetectionError,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return Image{}, sql.ErrNoRows
		}
		return Image{}, fmt.Errorf("failed to query or scan image info for %s: %w", originalPath, err)
	}
	return info, nil
}

// EnsureImageRecordExists creates a basic record if needed, setting tasks to pending.
// returns true if a new record was created, false otherwise.
func EnsureImageRecordExists(db Querier, originalPath string, modTime int64) (bool, error) {
	originalPath = filepath.ToSlash(originalPath)
	queryBuilder := psql.Insert("images").
		Columns("original_path", "last_modified", "thumbnail_status", "metadata_status", "detection_status").
		Values(originalPath, modTime, StatusPending, StatusPending, StatusPending).
		Suffix("ON CONFLICT(original_path) DO NOTHING")

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to build SQL query for EnsureImageRecordExists: %w", err)
	}

	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		return false, fmt.Errorf("failed to execute ensure image record for %s: %w", originalPath, err)
	}

	rowsAffected, _ := result.RowsAffected()
	created := rowsAffected > 0
	if created {
		log.Printf("database: Created initial image record for %s", originalPath)
	}
	return created, nil
}

// MarkImageTaskProcessing updates a task's status to processing
func MarkImageTaskProcessing(db Querier, originalPath, taskStatusColumn string) error {
	originalPath = filepath.ToSlash(originalPath)

	if taskStatusColumn != "thumbnail_status" && taskStatusColumn != "metadata_status" && taskStatusColumn != "detection_status" {
		return fmt.Errorf("invalid task status column name: %s", taskStatusColumn)
	}

	queryBuilder := psql.Update("images").
		Set(taskStatusColumn, StatusProcessing).
		// also clear any previous error for this task
		Set(strings.Replace(taskStatusColumn, "_status", "_error", 1), nil).
		Where(sq.Eq{"original_path": originalPath})

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL query for MarkImageTaskProcessing (%s): %w", taskStatusColumn, err)
	}

	_, err = db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to mark task %s processing for %s: %w", taskStatusColumn, originalPath, err)
	}
	return nil
}

// UpdateImageThumbnailResult updates thumbnail task outcome
func UpdateImageThumbnailResult(db Querier, originalPath string, thumbPath *string, modTime int64, taskErr error) error {
	originalPath = filepath.ToSlash(originalPath)
	now := time.Now().Unix()
	status := StatusDone
	var errStr *string
	if taskErr != nil {
		status = StatusError
		s := taskErr.Error()
		errStr = &s
	}

	queryBuilder := psql.Update("images").
		Set("thumbnail_path", thumbPath).
		Set("last_modified", modTime).
		Set("thumbnail_status", status).
		Set("thumbnail_processed_at", now).
		Set("thumbnail_error", errStr).
		Where(sq.Eq{"original_path": originalPath})

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL query for UpdateImageThumbnailResult: %w", err)
	}

	_, err = db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to update thumbnail result for %s: %w", originalPath, err)
	}
	return nil
}

// UpdateImageMetadataResult updates metadata task outcome
func UpdateImageMetadataResult(db Querier, originalPath string, meta *media.Metadata, modTime int64, taskErr error) error {
	originalPath = filepath.ToSlash(originalPath)
	now := time.Now().Unix()
	status := StatusDone
	var errStr *string
	if taskErr != nil {
		status = StatusError
		s := taskErr.Error()
		errStr = &s
		// don't save potentially partial metadata if error occurred? or save what we have?
		// here we save what we have (metadata might be non-nil even on error)
	}

	updateMap := map[string]interface{}{
		"last_modified":         modTime,
		"metadata_status":       status,
		"metadata_processed_at": now,
		"metadata_error":        errStr,
	}

	if meta != nil {
		updateMap["width"] = meta.Width
		updateMap["height"] = meta.Height
		updateMap["aperture"] = meta.Aperture
		updateMap["shutter_speed"] = meta.ShutterSpeed
		updateMap["iso"] = meta.ISO
		updateMap["focal_length"] = meta.FocalLength
		updateMap["lens_make"] = meta.LensMake
		updateMap["lens_model"] = meta.LensModel
		updateMap["camera_make"] = meta.CameraMake
		updateMap["camera_model"] = meta.CameraModel
		updateMap["taken_at"] = meta.TakenAt
	}

	queryBuilder := psql.Update("images").
		SetMap(updateMap).
		Where(sq.Eq{"original_path": originalPath})

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL query for UpdateImageMetadataResult: %w", err)
	}

	_, err = db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to update metadata result for %s: %w", originalPath, err)
	}
	return nil
}

// UpdateImageDetectionResult updates detection task outcome
func UpdateImageDetectionResult(db *sql.DB, originalPath string, detections []media.DetectionResult, modTime int64, taskErr error) error {
	originalPath = filepath.ToSlash(originalPath)
	now := time.Now().Unix()
	status := StatusDone
	var errStr *string
	if taskErr != nil {
		status = StatusError
		s := taskErr.Error()
		errStr = &s
		detections = nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for detection update: %w", err)
	}
	defer tx.Rollback()

	deletedCount, err := DeleteUntaggedFacesByImagePath(tx, originalPath)
	if err != nil {
		return fmt.Errorf("failed deleting old untagged faces in transaction for %s: %w", originalPath, err)
	}
	if deletedCount > 0 {
		log.Printf("database: Deleted %d old untagged faces for %s", deletedCount, originalPath)
	}

	addedCount := 0
	if taskErr == nil && len(detections) > 0 {
		for _, det := range detections {
			_, err := AddFace(tx, nil, originalPath, det.X, det.Y, det.X+det.W, det.Y+det.H)
			if err != nil {
				return fmt.Errorf("failed adding detected face in transaction for %s: %w", originalPath, err)
			}
			addedCount++
		}
		if addedCount > 0 {
			log.Printf("database: Added %d new untagged faces for %s", addedCount, originalPath)
		}
	}

	queryBuilder := psql.Update("images").
		Set("last_modified", modTime).
		Set("detection_status", status).
		Set("detection_processed_at", now).
		Set("detection_error", errStr).
		Where(sq.Eq{"original_path": originalPath})

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL query for UpdateImageDetectionResult: %w", err)
	}

	_, err = tx.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to update detection result for %s: %w", originalPath, err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit detection update transaction for %s: %w", originalPath, err)
	}

	return nil
}

func DeleteImageRecord(db Querier, originalPath string) error {
	originalPath = filepath.ToSlash(originalPath)

	queryBuilder := psql.Delete("images").Where(sq.Eq{"original_path": originalPath})
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL for DeleteImageRecord: %w", err)
	}
	_, err = db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to delete image record for %s: %w", originalPath, err)
	}
	return nil
}
