package database

const (
	// Filename
	SortFilenameAsc  = "filename_asc"
	SortFilenameDesc = "filename_desc"
	SortFilenameNat  = "filename_nat"

	// Capture time (EXIF TakenAt, falls back to file mod time)
	SortDateDesc = "date_desc"
	SortDateAsc  = "date_asc"

	// File system modification time
	SortModTimeDesc = "mod_time_desc"
	SortModTimeAsc  = "mod_time_asc"

	// File size
	SortFileSizeDesc = "file_size_desc"
	SortFileSizeAsc  = "file_size_asc"

	// Camera settings (EXIF)
	SortISOAsc           = "iso_asc"
	SortISODesc          = "iso_desc"
	SortApertureAsc      = "aperture_asc"
	SortApertureDesc     = "aperture_desc"
	SortFocalLengthAsc   = "focal_length_asc"
	SortFocalLengthDesc  = "focal_length_desc"
	SortShutterSpeedAsc  = "shutter_speed_asc"
	SortShutterSpeedDesc = "shutter_speed_desc"

	// Equipment (EXIF)
	SortCameraAsc = "camera_asc"
)

const DefaultSortOrder = SortFilenameAsc

func IsValidSortOrder(order string) bool {
	switch order {
	case SortFilenameAsc, SortFilenameDesc, SortFilenameNat,
		SortDateDesc, SortDateAsc,
		SortModTimeDesc, SortModTimeAsc,
		SortFileSizeDesc, SortFileSizeAsc,
		SortISOAsc, SortISODesc,
		SortApertureAsc, SortApertureDesc,
		SortFocalLengthAsc, SortFocalLengthDesc,
		SortShutterSpeedAsc, SortShutterSpeedDesc,
		SortCameraAsc:
		return true
	default:
		return false
	}
}

// basenameExpr extracts the filename component of images.original_path (which is
// stored as "<album folder>/<filename>", and can differ in folder prefix across
// albums when images are pulled together for a collection), so name-based sorts
// order on the filename rather than the full stored path.
const basenameExpr = "substring(original_path from '[^/]+$')"

// shutterSpeedSecondsExpr converts the stored shutter speed text (e.g. "1/125s" or
// "2s") to a numeric numbers-of-seconds value for ordering purposes. Non-numeric or
// unparsable values sort as NULL (last).
const shutterSpeedSecondsExpr = `(CASE
	WHEN shutter_speed IS NULL THEN NULL
	WHEN shutter_speed LIKE '%/%' THEN
		NULLIF(split_part(trim(trailing 's' from shutter_speed), '/', 1), '')::float8
		/ NULLIF(NULLIF(split_part(trim(trailing 's' from shutter_speed), '/', 2), '')::float8, 0)
	ELSE NULLIF(trim(trailing 's' from shutter_speed), '')::float8
END)`

// SQLOrderClause returns the ORDER BY expression (without the "ORDER BY" keywords)
// that implements the given sort order entirely in SQL. Every clause ends with a
// stable, non-null tiebreaker (original_path) so paging windows never overlap or
// skip rows when the primary sort key has duplicate or NULL values. Nullable
// numeric/EXIF columns always sort NULLs last, independent of direction.
func SQLOrderClause(order string) string {
	switch order {
	case SortFilenameDesc:
		return "LOWER(" + basenameExpr + ") DESC, original_path DESC"
	case SortFilenameNat:
		// approximate natural sort: shorter names first, then lexical. this
		// correctly orders runs like IMG_2.jpg < IMG_10.jpg as long as the
		// differing digits are the only length difference, which covers the
		// common camera-filename case without needing a natural sort collation.
		return "LENGTH(" + basenameExpr + ") ASC, LOWER(" + basenameExpr + ") ASC, original_path ASC"
	case SortDateDesc:
		return "COALESCE(taken_at, last_modified) DESC, original_path DESC"
	case SortDateAsc:
		return "COALESCE(taken_at, last_modified) ASC, original_path ASC"
	case SortModTimeDesc:
		return "last_modified DESC, original_path DESC"
	case SortModTimeAsc:
		return "last_modified ASC, original_path ASC"
	case SortFileSizeDesc:
		return "size DESC, original_path DESC"
	case SortFileSizeAsc:
		return "size ASC, original_path ASC"
	case SortISODesc:
		return "iso DESC NULLS LAST, original_path ASC"
	case SortISOAsc:
		return "iso ASC NULLS LAST, original_path ASC"
	case SortApertureDesc:
		return "aperture DESC NULLS LAST, original_path ASC"
	case SortApertureAsc:
		return "aperture ASC NULLS LAST, original_path ASC"
	case SortFocalLengthDesc:
		return "focal_length DESC NULLS LAST, original_path ASC"
	case SortFocalLengthAsc:
		return "focal_length ASC NULLS LAST, original_path ASC"
	case SortShutterSpeedDesc:
		return shutterSpeedSecondsExpr + " DESC NULLS LAST, original_path ASC"
	case SortShutterSpeedAsc:
		return shutterSpeedSecondsExpr + " ASC NULLS LAST, original_path ASC"
	case SortCameraAsc:
		return "LOWER(COALESCE(camera_make, '') || ' ' || COALESCE(camera_model, '')) ASC, original_path ASC"
	default:
		return "LOWER(" + basenameExpr + ") ASC, original_path ASC"
	}
}
