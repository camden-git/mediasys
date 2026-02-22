package media

import (
	"encoding/binary"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/rwcarlsen/goexif/exif"
)

// xmpRatingAttr matches xmp:Rating="N" attribute
var xmpRatingAttr = regexp.MustCompile(`xmp:Rating="(-?\d+)"`)

// xmpRatingElem matches <xmp:Rating>N</xmp:Rating> element
var xmpRatingElem = regexp.MustCompile(`<xmp:Rating>(-?\d+)</xmp:Rating>`)

// xmpSubjectSeq matches the dc:subject Seq/Bag block
var xmpSubjectSeq = regexp.MustCompile(`(?s)<dc:subject>\s*<rdf:(?:Seq|Bag)>(.*?)</rdf:(?:Seq|Bag)>\s*</dc:subject>`)

// xmpRdfLi matches individual <rdf:li> items
var xmpRdfLi = regexp.MustCompile(`<rdf:li[^>]*>([^<]+)</rdf:li>`)

// xmpNamespace is the XMP APP1 segment identifier string (with NUL terminator)
const xmpNamespace = "http://ns.adobe.com/xap/1.0/\x00"

// extractXMPRating reads JPEG APP1 segments looking for an embedded XMP block
// with an xmp:Rating value. Returns nil if the file is not a JPEG or has no rating.
func extractXMPRating(filePath string) *int {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	// check JPEG SOI marker
	soi := make([]byte, 2)
	if _, err := io.ReadFull(f, soi); err != nil {
		return nil
	}
	if soi[0] != 0xFF || soi[1] != 0xD8 {
		return nil // not a JPEG
	}

	buf := make([]byte, 4)
	for {
		if _, err := io.ReadFull(f, buf[:2]); err != nil {
			return nil
		}
		if buf[0] != 0xFF {
			return nil // lost sync
		}
		marker := buf[1]

		// SOS (Start of Scan) — compressed image data follows, stop
		if marker == 0xDA {
			return nil
		}
		// Standalone markers with no length field
		if marker == 0xD8 || marker == 0xD9 {
			continue
		}

		// read 2-byte segment length (includes the length field itself)
		if _, err := io.ReadFull(f, buf[:2]); err != nil {
			return nil
		}
		segLen := int(binary.BigEndian.Uint16(buf[:2]))
		if segLen < 2 {
			return nil
		}
		dataLen := segLen - 2

		if marker == 0xE1 && dataLen > len(xmpNamespace) {
			// APP1: could be XMP
			data := make([]byte, dataLen)
			if _, err := io.ReadFull(f, data); err != nil {
				return nil
			}
			if strings.HasPrefix(string(data), xmpNamespace) {
				xmp := string(data[len(xmpNamespace):])
				for _, re := range []*regexp.Regexp{xmpRatingAttr, xmpRatingElem} {
					if m := re.FindStringSubmatch(xmp); m != nil {
						n, err := strconv.Atoi(m[1])
						if err == nil && n >= 1 && n <= 5 {
							return &n
						}
					}
				}
			}
		} else {
			if _, err := f.Seek(int64(dataLen), io.SeekCurrent); err != nil {
				return nil
			}
		}
	}
}

// extractXMPKeywords reads JPEG APP1 XMP segments and extracts dc:subject keywords.
// Returns nil if the file is not a JPEG or has no keywords.
func extractXMPKeywords(filePath string) []string {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	soi := make([]byte, 2)
	if _, err := io.ReadFull(f, soi); err != nil {
		return nil
	}
	if soi[0] != 0xFF || soi[1] != 0xD8 {
		return nil
	}

	buf := make([]byte, 4)
	for {
		if _, err := io.ReadFull(f, buf[:2]); err != nil {
			return nil
		}
		if buf[0] != 0xFF {
			return nil
		}
		marker := buf[1]
		if marker == 0xDA {
			return nil
		}
		if marker == 0xD8 || marker == 0xD9 {
			continue
		}
		if _, err := io.ReadFull(f, buf[:2]); err != nil {
			return nil
		}
		segLen := int(binary.BigEndian.Uint16(buf[:2]))
		if segLen < 2 {
			return nil
		}
		dataLen := segLen - 2

		if marker == 0xE1 && dataLen > len(xmpNamespace) {
			data := make([]byte, dataLen)
			if _, err := io.ReadFull(f, data); err != nil {
				return nil
			}
			if strings.HasPrefix(string(data), xmpNamespace) {
				xmp := string(data[len(xmpNamespace):])
				seqMatch := xmpSubjectSeq.FindStringSubmatch(xmp)
				if seqMatch == nil {
					return nil
				}
				items := xmpRdfLi.FindAllStringSubmatch(seqMatch[1], -1)
				if len(items) == 0 {
					return nil
				}
				keywords := make([]string, 0, len(items))
				for _, item := range items {
					keywords = append(keywords, strings.TrimSpace(item[1]))
				}
				return keywords
			}
		} else {
			if _, err := f.Seek(int64(dataLen), io.SeekCurrent); err != nil {
				return nil
			}
		}
	}
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
		return nil // Cannot represent as a fraction
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
		s := fmt.Sprintf("%.4fs", val) // use float representation if not a simple fraction
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
		// not necessarily a fatal error, the file might just lack EXIF data
		log.Printf("metadata: No EXIF data found or error decoding EXIF for %s: %v", filePath, err)
		// return metadata struct with only dimensions and any XMP data
		return &Metadata{Width: width, Height: height, Rating: extractXMPRating(filePath), Keywords: extractXMPKeywords(filePath)}, nil
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

	meta.Rating = extractXMPRating(filePath)
	meta.Keywords = extractXMPKeywords(filePath)

	return meta, nil
}
