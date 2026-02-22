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
