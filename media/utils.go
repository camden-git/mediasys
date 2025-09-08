package media

// minFloat32 returns the minimum of two float32 values
func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// maxFloat32 returns the maximum of two float32 values
func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// minInt returns the minimum of two int values
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt returns the maximum of two int values
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
