package media

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestFormatShutterSpeed(t *testing.T) {
	cases := []struct {
		num, den int64
		want     string
	}{
		{1, 250, "1/250"},
		{10, 600, "1/60"},
		{1, 2, "1/2"},
		{4, 5, "0.8s"},
		{30, 1, "30.0s"},
	}
	for _, c := range cases {
		got := formatShutterSpeed(c.num, c.den)
		if got == nil || *got != c.want {
			t.Errorf("formatShutterSpeed(%d,%d) = %v, want %s", c.num, c.den, got, c.want)
		}
	}
	if formatShutterSpeed(0, 1) != nil || formatShutterSpeed(1, 0) != nil {
		t.Error("expected nil for zero numerator/denominator")
	}
}

func jpegWithXMP(xmp string, fill int) []byte {
	var b bytes.Buffer
	b.Write([]byte{0xFF, 0xD8})
	// APP0 segment first, then optional 0xFF fill bytes before the XMP marker
	b.Write([]byte{0xFF, 0xE0, 0x00, 0x04, 0x00, 0x00})
	for i := 0; i < fill; i++ {
		b.WriteByte(0xFF)
	}
	payload := append([]byte(xmpNamespace), xmp...)
	b.Write([]byte{0xFF, 0xE1})
	_ = binary.Write(&b, binary.BigEndian, uint16(len(payload)+2))
	b.Write(payload)
	b.Write([]byte{0xFF, 0xDA})
	return b.Bytes()
}

func TestXMPParsing(t *testing.T) {
	const packet = `<x:xmpmeta><rdf:Description xmp:Rating='4'>` +
		`<dc:subject><rdf:Bag><rdf:li>Tom &amp; Jerry</rdf:li><rdf:li>team/a &lt;b&gt;</rdf:li></rdf:Bag></dc:subject>` +
		`</rdf:Description></x:xmpmeta>`

	for _, fill := range []int{0, 3} {
		xmp := readXMPPacket(bytes.NewReader(jpegWithXMP(packet, fill)))
		if xmp == "" {
			t.Fatalf("fill=%d: no XMP packet found", fill)
		}
		if r := xmpRating(xmp); r == nil || *r != 4 {
			t.Errorf("fill=%d: rating = %v, want 4", fill, r)
		}
		kws := xmpKeywords(xmp)
		if len(kws) != 2 || kws[0] != "Tom & Jerry" || kws[1] != "team/a <b>" {
			t.Errorf("fill=%d: keywords = %q", fill, kws)
		}
	}

	if r := xmpRating(`<x xmp:Rating="-1"/>`); r != nil {
		t.Errorf("rejected rating should be nil, got %d", *r)
	}
}
