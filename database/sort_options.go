package database

const (
	SortFilenameAsc = "filename_asc"
	SortFilenameNat = "filename_nat"
	SortDateDesc    = "date_desc"
	SortDateAsc     = "date_asc"
)

const DefaultSortOrder = SortFilenameAsc

// IsValidSortOrder checks if a string is a valid sort order constant
func IsValidSortOrder(order string) bool {
	switch order {
	case SortFilenameAsc, SortDateDesc, SortDateAsc, SortFilenameNat:
		return true
	default:
		return false
	}
}
