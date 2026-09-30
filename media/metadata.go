package media

import (
	"encoding/binary"
	"fmt"
	"html"
	"image"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

// xmpRatingAttr matches xmp:Rating="N" attribute
var xmpRatingAttr = regexp.MustCompile(`xmp:Rating=["'](-?\d+)["']`)

// xmpRatingElem matches <xmp:Rating>N</xmp:Rating> element
var xmpRatingElem = regexp.MustCompile(`<xmp:Rating>(-?\d+)</xmp:Rating>`)

// xmpSubjectSeq matches the dc:subject Seq/Bag block
var xmpSubjectSeq = regexp.MustCompile(`(?s)<dc:subject>\s*<rdf:(?:Seq|Bag)>(.*?)</rdf:(?:Seq|Bag)>\s*</dc:subject>`)

// xmpRdfLi matches individual <rdf:li> items
var xmpRdfLi = regexp.MustCompile(`<rdf:li[^>]*>([^<]+)</rdf:li>`)

// xmpNamespace is the XMP APP1 segment identifier string (with NUL terminator)
const xmpNamespace = "http://ns.adobe.com/xap/1.0/\x00"

// readXMPPacket walks the JPEG segments of r looking for the XMP APP1 block and
// returns its XML. It returns "" for non-JPEG input or when there is no XMP.
func readXMPPacket(r io.ReadSeeker) string {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return ""
	}
	var hdr [2]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil || hdr[0] != 0xFF || hdr[1] != 0xD8 {
		return "" // not a JPEG
	}

	for {
		// a marker is 0xFF, optionally preceded by any number of 0xFF fill bytes
		if _, err := io.ReadFull(r, hdr[:1]); err != nil || hdr[0] != 0xFF {
			return ""
		}
		var marker byte
		for {
			if _, err := io.ReadFull(r, hdr[:1]); err != nil {
				return ""
			}
			if hdr[0] != 0xFF {
				marker = hdr[0]
				break
			}
		}

		switch {
		case marker == 0xDA: // SOS: compressed data follows, no more metadata segments
			return ""
		case marker == 0x00, marker == 0x01, marker >= 0xD0 && marker <= 0xD9:
			continue // standalone markers without a length field
		}

		if _, err := io.ReadFull(r, hdr[:]); err != nil {
			return ""
		}
		segLen := int(binary.BigEndian.Uint16(hdr[:]))
		if segLen < 2 {
			return ""
		}
		dataLen := segLen - 2

		if marker == 0xE1 && dataLen > len(xmpNamespace) {
			data := make([]byte, dataLen)
			if _, err := io.ReadFull(r, data); err != nil {
				return ""
			}
			if strings.HasPrefix(string(data[:len(xmpNamespace)]), xmpNamespace) {
				return string(data[len(xmpNamespace):])
			}
		} else if _, err := r.Seek(int64(dataLen), io.SeekCurrent); err != nil {
			return ""
		}
	}
}

// xmpRating returns the star rating from an XMP packet. Unrated (0) and
// rejected (-1) images have no rating.
func xmpRating(xmp string) *int {
	for _, re := range []*regexp.Regexp{xmpRatingAttr, xmpRatingElem} {
		if m := re.FindStringSubmatch(xmp); m != nil {
			n, err := strconv.Atoi(m[1])
			if err == nil && n >= 1 && n <= 5 {
				return &n
			}
			return nil
		}
	}
	return nil
}

// xmpKeywords returns the unescaped dc:subject keywords from an XMP packet.
func xmpKeywords(xmp string) []string {
	seqMatch := xmpSubjectSeq.FindStringSubmatch(xmp)
	if seqMatch == nil {
		return nil
	}
	items := xmpRdfLi.FindAllStringSubmatch(seqMatch[1], -1)
	keywords := make([]string, 0, len(items))
	for _, item := range items {
		if kw := strings.TrimSpace(html.UnescapeString(item[1])); kw != "" {
			keywords = append(keywords, kw)
		}
	}
	if len(keywords) == 0 {
		return nil
	}
	return keywords
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

	return formatShutterSpeed(num, den)
}

// formatShutterSpeed renders an exposure time: fractions of a second as 1/N
// (10/600 -> 1/60), longer exposures in seconds.
func formatShutterSpeed(num, den int64) *string {
	if num <= 0 || den <= 0 {
		return nil
	}
	val := float64(num) / float64(den)
	if val <= 0.5 {
		s := fmt.Sprintf("1/%d", int(math.Round(1/val)))
		return &s
	}
	s := fmt.Sprintf("%.1fs", val) // e.g. 0.8s, 1.5s, 30.0s
	return &s
}

// exifDateLayout is the EXIF date/time format (no zone information).
const exifDateLayout = "2006:01:02 15:04:05"

// takenAt reads the capture time. EXIF stores camera wall-clock time without a
// reliable zone, and the frontend displays it in UTC, so the wall-clock value is
// stored as if it were UTC; OffsetTimeOriginal is deliberately ignored.
func takenAt(exifData *exif.Exif) *int64 {
	for _, field := range []exif.FieldName{exif.DateTimeOriginal, exif.DateTimeDigitized, exif.DateTime} {
		tag, err := exifData.Get(field)
		if err != nil || tag == nil {
			continue
		}
		raw, err := tag.StringVal()
		if err != nil {
			continue
		}
		t, err := time.ParseInLocation(exifDateLayout, strings.Trim(raw, "\x00 "), time.UTC)
		if err != nil {
			continue
		}
		ts := t.Unix()
		return &ts
	}
	return nil
}

// GetImageMetadata extracts relevant metadata using goexif
func GetImageMetadata(filePath string) (*Metadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("metadata: failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	var width, height *int
	if config, _, err := image.DecodeConfig(file); err == nil {
		w, h := config.Width, config.Height
		width, height = &w, &h
	}

	xmp := readXMPPacket(file)
	meta := &Metadata{Width: width, Height: height, Rating: xmpRating(xmp), Keywords: xmpKeywords(xmp)}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("metadata: failed to seek file %s: %w", filePath, err)
	}
	exifData, err := exif.Decode(file)
	if err != nil {
		// not necessarily a fatal error, the file might just lack EXIF data
		return meta, nil
	}

	meta.Aperture = getRational(exifData, exif.FNumber)
	meta.ShutterSpeed = getShutterSpeed(exifData)
	meta.ISO = getInt(exifData, exif.ISOSpeedRatings)
	meta.FocalLength = getRational(exifData, exif.FocalLength)
	meta.LensMake = getString(exifData, exif.LensMake)
	meta.LensModel = getString(exifData, exif.LensModel)
	meta.CameraMake = getString(exifData, exif.Make)
	meta.CameraModel = getString(exifData, exif.Model)
	meta.TakenAt = takenAt(exifData)
	return meta, nil
}
