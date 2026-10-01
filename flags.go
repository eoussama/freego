package freego

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// The bitfields below may carry undocumented internal bits, which are
// preserved as-is when decoding. Negative values, which JavaScript bitwise
// operations produce when bit 31 is set, are read as their unsigned 32-bit
// equivalent.

// UnmarshalJSON implements [json.Unmarshaler].
func (f *ProductFlags) UnmarshalJSON(data []byte) error { return decodeBits(data, (*uint64)(f)) }

// UnmarshalJSON implements [json.Unmarshaler].
func (f *ImageFlags) UnmarshalJSON(data []byte) error { return decodeBits(data, (*uint64)(f)) }

// UnmarshalJSON implements [json.Unmarshaler].
func (f *URLFlags) UnmarshalJSON(data []byte) error { return decodeBits(data, (*uint64)(f)) }

// TextFlags is an undocumented bitfield of a [LocalizedText].
type TextFlags uint64

// UnmarshalJSON implements [json.Unmarshaler].
func (f *TextFlags) UnmarshalJSON(data []byte) error { return decodeBits(data, (*uint64)(f)) }

// decodeBits decodes a JSON number (or null, as 0) into a bitfield.
func decodeBits(data []byte, out *uint64) error {
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("freego: invalid flags %s: %w", data, err)
	}
	v, err := numberToInt64(n)
	if err != nil {
		return fmt.Errorf("freego: invalid flags: %w", err)
	}
	if v < 0 && v >= math.MinInt32 {
		*out = uint64(uint32(int32(v)))
	} else {
		*out = uint64(v)
	}
	return nil
}

// ProductFlags is a bitfield of product properties.
type ProductFlags uint64

// Known product flags.
const (
	// ProductFlagTrash marks low quality products, which FreeStuff's own
	// bot hides when its "low quality" filter is enabled.
	ProductFlagTrash ProductFlags = 1 << 0
	// ProductFlagThirdParty marks offers made through a third-party site.
	// Deprecated by FreeStuff.
	ProductFlagThirdParty ProductFlags = 1 << 1
	// ProductFlagPermanent marks permanent offers.
	ProductFlagPermanent ProductFlags = 1 << 2
	// ProductFlagStaffPick marks offers picked by FreeStuff's staff.
	ProductFlagStaffPick ProductFlags = 1 << 3
	// ProductFlagFirstPartyExclusive marks offers reserved for FreeStuff's
	// official channels. They should never be received; ignore them if they
	// are (see [Product.ShouldIgnore]).
	ProductFlagFirstPartyExclusive ProductFlags = 1 << 4
)

// Has reports whether every bit of flag is set.
func (f ProductFlags) Has(flag ProductFlags) bool { return flag != 0 && f&flag == flag }

// String lists the set flags, e.g. "TRASH|STAFF_PICK".
func (f ProductFlags) String() string {
	return flagString(uint64(f), []string{"TRASH", "THIRDPARTY", "PERMANENT", "STAFF_PICK", "FIRSTPARTY_EXCLUSIVE"})
}

// ImageFlags is a bitfield of image properties.
type ImageFlags uint64

// Known image flags. The AR_ flags give the aspect ratio, TP_ the type of
// image and FT_ its features.
const (
	ImageFlagProxied     ImageFlags = 1 << 0 // proxied and/or provided by FreeStuff
	ImageFlagWide        ImageFlags = 1 << 1 // landscape orientation
	ImageFlagSquare      ImageFlags = 1 << 2 // square
	ImageFlagTall        ImageFlags = 1 << 3 // portrait orientation
	ImageFlagPromo       ImageFlags = 1 << 4 // promotional image
	ImageFlagLogo        ImageFlags = 1 << 5 // the product's logo
	ImageFlagShowcase    ImageFlags = 1 << 6 // showcase, e.g. in-game screenshot
	ImageFlagOther       ImageFlags = 1 << 7 // any other kind of image
	ImageFlagWatermarked ImageFlags = 1 << 8 // has a FreeStuff watermark
	ImageFlagTags        ImageFlags = 1 << 9 // has the tags rendered onto it
)

// Has reports whether every bit of flag is set.
func (f ImageFlags) Has(flag ImageFlags) bool { return flag != 0 && f&flag == flag }

// String lists the set flags, e.g. "PROXIED|AR_WIDE".
func (f ImageFlags) String() string {
	return flagString(uint64(f), []string{"PROXIED", "AR_WIDE", "AR_SQUARE", "AR_TALL", "TP_PROMO", "TP_LOGO", "TP_SHOWCASE", "TP_OTHER", "FT_WATERMARK", "FT_TAGS"})
}

// URLFlags is a bitfield of URL properties.
type URLFlags uint64

// Known URL flags.
const (
	URLFlagOriginal       URLFlags = 1 << 0 // the original link, unmodified
	URLFlagProxied        URLFlags = 1 << 1 // proxied through FreeStuff or a third party
	URLFlagTracking       URLFlags = 1 << 2 // counts clicks; no personal data collected
	URLFlagOpensInBrowser URLFlags = 1 << 3 // opens the store's website
	URLFlagOpensInClient  URLFlags = 1 << 4 // opens the store's client application
)

// Has reports whether every bit of flag is set.
func (f URLFlags) Has(flag URLFlags) bool { return flag != 0 && f&flag == flag }

// String lists the set flags, e.g. "ORIGINAL|OPENS_IN_BROWSER".
func (f URLFlags) String() string {
	return flagString(uint64(f), []string{"ORIGINAL", "PROXIED", "TRACKING", "OPENS_IN_BROWSER", "OPENS_IN_CLIENT"})
}

// flagString renders the set bits of v using names for the low bits and hex
// for any remaining unknown bits.
func flagString(v uint64, names []string) string {
	if v == 0 {
		return "0"
	}
	var parts []string
	for i, name := range names {
		if v&(1<<i) != 0 {
			parts = append(parts, name)
			v &^= 1 << i
		}
	}
	if v != 0 {
		parts = append(parts, "0x"+strconv.FormatUint(v, 16))
	}
	return strings.Join(parts, "|")
}
