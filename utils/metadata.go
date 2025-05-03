package utils

import (
	"fmt"
	"image"
	"log"
	"os"
	"strings"

	"github.com/rwcarlsen/goexif/exif"
)

type Metadata struct {
	Width        *int     `json:"width,omitempty"` // get from DecodeConfig usually
	Height       *int     `json:"height,omitempty"`
	Aperture     *float64 `json:"aperture,omitempty"`
	ShutterSpeed *string  `json:"shutter_speed,omitempty"`
	ISO          *int     `json:"iso,omitempty"`
	FocalLength  *float64 `json:"focal_length,omitempty"`
	LensMake     *string  `json:"lens_make,omitempty"`
	LensModel    *string  `json:"lens_model,omitempty"`
	CameraMake   *string  `json:"camera_make,omitempty"`
	CameraModel  *string  `json:"camera_model,omitempty"`
	TakenAt      *int64   `json:"taken_at,omitempty"`
}

// helper to safely get and convert a rational tag (like Aperture, FocalLength)
func getRational(exifData *exif.Exif, tagName exif.FieldName) *float64 {
	tag, err := exifData.Get(tagName)
	if err != nil || tag == nil {
		return nil // Tag not found
	}
	// rational numbers are often stored as num/den
	num, den, err := tag.Rat2(0)
	if err != nil || den == 0 {
		// sometimes stored as Int instead
		valInt, errInt := tag.Int(0)
		if errInt == nil {
			fVal := float64(valInt)
			return &fVal
		}
		return nil
	}
	val := float64(num) / float64(den)
	return &val
}

// helper to safely get and convert an integer tag (like ISO)
func getInt(exifData *exif.Exif, tagName exif.FieldName) *int {
	tag, err := exifData.Get(tagName)
	if err != nil || tag == nil {
		return nil
	}
	// ISO might be a slice, get the first value
	val, err := tag.Int(0)
	if err != nil {
		// log.Printf("Error converting int tag %s: %v", tagName, err)
		return nil
	}
	return &val
}

// helper to safely get a string tag, trimming null terminators
func getString(exifData *exif.Exif, tagName exif.FieldName) *string {
	tag, err := exifData.Get(tagName)
	if err != nil || tag == nil {
		return nil
	}
	// val string might have null chars at the end
	val := strings.TrimRight(tag.String(), "\x00")
	if val == "" {
		return nil
	}
	return &val
}

// helper to get Shutter Speed specifically, formatting it nicely
func getShutterSpeed(exifData *exif.Exif) *string {
	tag, err := exifData.Get(exif.ExposureTime)
	if err != nil || tag == nil {
		return nil
	}
	num, den, err := tag.Rat2(0)
	if err != nil || den == 0 {
		return nil // Cannot represent as fraction
	}

	if num == 1 && den > 1 { // common case: 1/XXX
		s := fmt.Sprintf("1/%d", den)
		return &s
	}

	// handle cases like 1/2.5 -> 1/3 or 1/2
	val := float64(num) / float64(den)
	if val >= 1.0 {
		s := fmt.Sprintf("%.1fs", val) // e.g., 1.5s, 30.0s
		return &s
	} else {
		s := fmt.Sprintf("%.4fs", val) // use float representation if not simple fraction
		return &s
	}
}

// GetImageMetadata extracts relevant metadata using goexif
func GetImageMetadata(filePath string) (*Metadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("metadata: failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	config, format, err := image.DecodeConfig(file)
	var width, height *int
	if err == nil {
		w, h := config.Width, config.Height
		width = &w
		height = &h
		log.Printf("metadata: Decoded dimensions for %s (format: %s): %dx%d", filePath, format, *width, *height)
	} else {
		log.Printf("metadata: Warning - Could not decode config for dimensions of %s: %v", filePath, err)
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("metadata: failed to seek file %s: %w", filePath, err)
	}

	exifData, err := exif.Decode(file)
	if err != nil {
		// not necessarily a fatal error, file might just lack EXIF data
		log.Printf("metadata: No EXIF data found or error decoding EXIF for %s: %v", filePath, err)
		// return metadata struct with only dimensions if they were found
		return &Metadata{Width: width, Height: height}, nil
	}

	meta := &Metadata{
		Width:        width,
		Height:       height,
		Aperture:     getRational(exifData, exif.FNumber),
		ShutterSpeed: getShutterSpeed(exifData),
		ISO:          getInt(exifData, exif.ISOSpeedRatings),
		FocalLength:  getRational(exifData, exif.FocalLength),
		LensMake:     getString(exifData, exif.LensMake),
		LensModel:    getString(exifData, exif.LensModel),
		CameraMake:   getString(exifData, exif.Make),
		CameraModel:  getString(exifData, exif.Model),
	}

	dt, err := exifData.DateTime()
	if err == nil {
		ts := dt.Unix()
		meta.TakenAt = &ts
	} else {
		log.Printf("metadata: Could not read DateTimeOriginal for %s: %v", filePath, err)
	}

	return meta, nil
}
