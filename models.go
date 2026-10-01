package freego

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Product is a free game, DLC, in-game item or anything else FreeStuff
// announces. Fields follow the data model of [CompatibilityDate].
type Product struct {
	ID     int64          `json:"id"`
	Title  string         `json:"title"`
	Prices []ProductPrice `json:"prices"`
	Kind   ProductKind    `json:"kind"`
	// Tags are unsorted, non-standardized, human readable tags.
	Tags []string `json:"tags"`
	// Images only holds a subset of all available images on the free tier.
	Images      []ProductImage `json:"images"`
	Description Descriptions   `json:"description"`
	// Rating is an aggregated score from 0 (worst) to 1 (best).
	Rating float64 `json:"rating"`
	// Copyright is the name of the IP's copyright holder.
	Copyright string `json:"copyright"`
	// Until is when the offer ends; zero when it has no known end.
	Until Time `json:"until"`
	// Type is the channel the product was announced in.
	Type Channel `json:"type"`
	// URLs only holds a subset of all available URLs on the free tier.
	URLs  []ProductURL `json:"urls"`
	Store Store        `json:"store"`
	// Platforms is always empty on the free tier.
	Platforms []Platform   `json:"platforms"`
	Flags     ProductFlags `json:"flags"`
	// Notice is a rare, optional note from FreeStuff's team meant to be
	// displayed alongside the product.
	Notice string `json:"notice"`
	// Meta is always empty on the free tier. See [Product.MetaValue].
	Meta []ProductMeta `json:"meta"`
	// StaffApproved reports whether a human checked the data before it was
	// published.
	StaffApproved bool `json:"staffApproved"`
	// UpdatedAt identifies the product's last update. FreeStuff does not
	// document its unit, so it is passed through unchanged.
	UpdatedAt int64 `json:"updatedAt"`
}

// UnmarshalJSON decodes a product. The API types every numeric field as a
// JSON number, so ID and UpdatedAt are also accepted in exponent or
// non-integral notation rather than failing the whole product.
func (p *Product) UnmarshalJSON(data []byte) error {
	type plain Product
	aux := struct {
		*plain
		ID        json.Number `json:"id"`
		UpdatedAt json.Number `json:"updatedAt"`
	}{plain: (*plain)(p)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	if p.ID, err = numberToInt64(aux.ID); err != nil {
		return fmt.Errorf("freego: product id: %w", err)
	}
	if p.UpdatedAt, err = numberToInt64(aux.UpdatedAt); err != nil {
		return fmt.Errorf("freego: product updatedAt: %w", err)
	}
	return nil
}

// numberToInt64 converts a JSON number to an int64, rounding non-integral
// values. An empty number (from null or an absent field) is 0.
func numberToInt64(n json.Number) (int64, error) {
	if n == "" {
		return 0, nil
	}
	if i, err := strconv.ParseInt(string(n), 10, 64); err == nil {
		return i, nil
	}
	f, err := strconv.ParseFloat(string(n), 64)
	if err != nil || math.IsNaN(f) || f >= 0x1p63 || f < -0x1p63 {
		return 0, fmt.Errorf("invalid integer %q", n)
	}
	return int64(math.Round(f)), nil
}

// ShouldIgnore reports whether FreeStuff asks API consumers not to use this
// product, which is the case when it carries
// [ProductFlagFirstPartyExclusive].
func (p *Product) ShouldIgnore() bool {
	return p.Flags.Has(ProductFlagFirstPartyExclusive)
}

// MetaValue returns the value of the first meta entry with the given key (see
// the Meta* constants). Values are JSON-compatible strings such as "true",
// "12" or "-4e8".
func (p *Product) MetaValue(key string) (string, bool) {
	for _, m := range p.Meta {
		if m.Key == key {
			return m.Value, true
		}
	}
	return "", false
}

// Price returns the price in the given currency code (e.g. "USD"), compared
// case-insensitively.
func (p *Product) Price(currency string) (ProductPrice, bool) {
	for _, pr := range p.Prices {
		if strings.EqualFold(pr.Currency, currency) {
			return pr, true
		}
	}
	return ProductPrice{}, false
}

// BestImage returns the image best suited for presentation, which is the
// one with the highest priority, optionally restricted to images having every
// flag in want (pass 0 for any image).
func (p *Product) BestImage(want ImageFlags) (ProductImage, bool) {
	var best ProductImage
	found := false
	for _, img := range p.Images {
		if want != 0 && !img.Flags.Has(want) {
			continue
		}
		if !found || img.Priority > best.Priority {
			best, found = img, true
		}
	}
	return best, found
}

// BestURL returns the URL best suited for use, which is the one with the
// highest priority, optionally restricted to URLs having every flag in want
// (pass 0 for any URL).
func (p *Product) BestURL(want URLFlags) (ProductURL, bool) {
	var best ProductURL
	found := false
	for _, u := range p.URLs {
		if want != 0 && !u.Flags.Has(want) {
			continue
		}
		if !found || u.Priority > best.Priority {
			best, found = u, true
		}
	}
	return best, found
}

// ProductPrice is a product's price in one currency. Values are in the
// currency's smallest unit: 499 in USD is $4.99.
type ProductPrice struct {
	// Currency is an ISO 4217 currency code.
	Currency string `json:"currency"`
	// OldValue is the price before the discount.
	OldValue int64 `json:"oldValue"`
	// NewValue is the price after the discount, usually 0.
	NewValue int64 `json:"newValue"`
	// Converted reports whether the value was converted from the product's
	// EUR or USD price using market rates, rather than taken from the store.
	Converted bool `json:"converted"`
}

// UnmarshalJSON decodes a price, rounding non-integral values instead of
// failing on them.
func (pp *ProductPrice) UnmarshalJSON(data []byte) error {
	var aux struct {
		Currency  string      `json:"currency"`
		OldValue  json.Number `json:"oldValue"`
		NewValue  json.Number `json:"newValue"`
		Converted bool        `json:"converted"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	oldValue, err := numberToInt64(aux.OldValue)
	if err != nil {
		return fmt.Errorf("freego: price oldValue: %w", err)
	}
	newValue, err := numberToInt64(aux.NewValue)
	if err != nil {
		return fmt.Errorf("freego: price newValue: %w", err)
	}
	*pp = ProductPrice{Currency: aux.Currency, OldValue: oldValue, NewValue: newValue, Converted: aux.Converted}
	return nil
}

// ProductImage is an image of a product.
type ProductImage struct {
	URL   string     `json:"url"`
	Flags ImageFlags `json:"flags"`
	// Priority tells how well the image is suited for presentation; higher
	// is better.
	Priority float64 `json:"priority"`
}

// ProductURL is a link to a product.
type ProductURL struct {
	URL   string   `json:"url"`
	Flags URLFlags `json:"flags"`
	// Priority tells how well the URL is suited for use; higher is better.
	Priority  float64    `json:"priority"`
	Store     Store      `json:"store"`
	Platforms []Platform `json:"platforms"`
}

// ProductMeta is a metadata entry of a product.
type ProductMeta struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// LocalizedText is a text in one language.
type LocalizedText struct {
	// Lang is the language of the text.
	Lang string `json:"lang"`
	Text string `json:"text"`
	// Priority ranks the entries; higher is better.
	Priority float64 `json:"priority"`
	// Flags is an undocumented bitfield.
	Flags TextFlags `json:"flags"`
}

// Descriptions holds a product's description in one or more languages.
//
// Older compatibility dates deliver the description as a single English
// string; it decodes to one entry with Lang "en".
type Descriptions []LocalizedText

// UnmarshalJSON accepts either a list of localized texts or a plain string.
func (d *Descriptions) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	switch {
	case bytes.Equal(data, []byte("null")):
		*d = nil
		return nil
	case len(data) > 0 && data[0] == '"':
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		if s == "" {
			*d = nil
		} else {
			*d = Descriptions{{Lang: "en", Text: s}}
		}
		return nil
	}
	var list []LocalizedText
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	*d = list
	return nil
}

// Get returns the description in the given language, such as "de" or
// "en-US". An exact match is preferred, then one with the same base language
// (so "en" matches "en-US" and the other way around). When no entry matches,
// the [Descriptions.Default] text is returned.
func (d Descriptions) Get(lang string) string {
	base := baseLang(lang)
	var partial *LocalizedText
	for i := range d {
		if strings.EqualFold(d[i].Lang, lang) {
			return d[i].Text
		}
		if partial == nil && strings.EqualFold(baseLang(d[i].Lang), base) {
			partial = &d[i]
		}
	}
	if partial != nil {
		return partial.Text
	}
	return d.Default()
}

// Default returns the English description if there is one, and otherwise
// the one with the highest priority. It returns "" when there is no
// description.
func (d Descriptions) Default() string {
	var best *LocalizedText
	for i := range d {
		if strings.EqualFold(baseLang(d[i].Lang), "en") {
			return d[i].Text
		}
		if best == nil || d[i].Priority > best.Priority {
			best = &d[i]
		}
	}
	if best == nil {
		return ""
	}
	return best.Text
}

func baseLang(lang string) string {
	if i := strings.IndexAny(lang, "-_"); i >= 0 {
		return lang[:i]
	}
	return lang
}

// Time is a point in time sent by FreeStuff as a Unix timestamp in
// milliseconds. The zero value means "not set"; check it with IsZero.
type Time struct {
	time.Time
}

// UnmarshalJSON accepts a number of milliseconds since the Unix epoch, a
// numeric string, an RFC 3339 string, or null. 0 and null decode to the zero
// Time.
func (t *Time) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		*t = Time{}
		return nil
	}

	s := string(data)
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		if s == "" {
			*t = Time{}
			return nil
		}
		if parsed, err := time.Parse(time.RFC3339Nano, s); err == nil {
			*t = Time{parsed.UTC()}
			return nil
		}
	}

	// Bound to the years 0000-9999, which also rejects NaN and infinities.
	const maxMillis = 253402300799999
	ms, err := strconv.ParseFloat(s, 64)
	if err != nil || !(ms >= -62167219200000 && ms <= maxMillis) {
		return fmt.Errorf("freego: invalid timestamp %s", data)
	}
	if ms == 0 {
		*t = Time{}
		return nil
	}
	*t = Time{time.UnixMilli(int64(ms)).UTC()}
	return nil
}

// MarshalJSON encodes the time as milliseconds since the Unix epoch, or 0 for
// the zero Time, matching the API's format.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("0"), nil
	}
	return []byte(strconv.FormatInt(t.UnixMilli(), 10)), nil
}

// Announcement is a set of products published together. All of its products
// share the same channel.
type Announcement struct {
	ID int64 `json:"id"`
	// Products holds the ids of the products in the announcement.
	Products []int64 `json:"products"`
}

// ResolvedAnnouncement is an [Announcement] with its products included, as
// delivered by the announcement_created webhook event.
type ResolvedAnnouncement struct {
	Announcement
	ResolvedProducts []Product `json:"resolvedProducts"`
}
