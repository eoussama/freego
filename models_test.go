package freego

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func loadProduct(t *testing.T, name string) Product {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	var p Product
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("decoding %s: %v", name, err)
	}
	return p
}

func TestProductDecodeCurrentFormat(t *testing.T) {
	p := loadProduct(t, "product_2026-06-08.json")

	if p.ID != 123456 || p.Title != "Example Game" || p.Kind != KindGame || p.Type != ChannelKeep || p.Store != StoreSteam {
		t.Errorf("unexpected scalar fields: %+v", p)
	}
	if p.Rating != 0.87 || p.Copyright != "Example Studio" || !p.StaffApproved || p.UpdatedAt != 1790860000000 {
		t.Errorf("unexpected fields: rating=%v copyright=%q approved=%v updatedAt=%d", p.Rating, p.Copyright, p.StaffApproved, p.UpdatedAt)
	}
	if want := time.UnixMilli(1790867606000).UTC(); !p.Until.Equal(want) {
		t.Errorf("Until = %v, want %v", p.Until, want)
	}
	if len(p.Prices) != 2 || p.Prices[0] != (ProductPrice{Currency: "USD", OldValue: 1999}) || !p.Prices[1].Converted {
		t.Errorf("unexpected prices: %+v", p.Prices)
	}
	if len(p.Description) != 2 || p.Description[0].Lang != "de" || p.Description[1].Priority != 2 {
		t.Errorf("unexpected description: %+v", p.Description)
	}
	if len(p.URLs) != 2 || p.URLs[0].Store != StoreSteam || len(p.URLs[0].Platforms) != 2 {
		t.Errorf("unexpected urls: %+v", p.URLs)
	}
	// Unknown enum values and flag bits are preserved.
	if p.Platforms[2] != Platform("quantum") {
		t.Errorf("unknown platform not preserved: %v", p.Platforms)
	}
	if p.Flags != ProductFlagPermanent|ProductFlagStaffPick|1<<12 {
		t.Errorf("Flags = %v", p.Flags)
	}
}

func TestProductDecodeOlderFormat(t *testing.T) {
	p := loadProduct(t, "product_2025-03-01.json")

	if got := p.Description.Get("en"); got != "A description delivered as a plain string." {
		t.Errorf("Description.Get = %q", got)
	}
	if !p.Until.IsZero() {
		t.Errorf("Until = %v, want zero", p.Until)
	}
	if len(p.URLs) != 1 || p.URLs[0].Store != "" || p.URLs[0].Platforms != nil {
		t.Errorf("unexpected urls: %+v", p.URLs)
	}
	if p.Notice != "Only in some regions" || p.Kind != KindDLC || p.Type != ChannelTimed {
		t.Errorf("unexpected fields: %+v", p)
	}
}

func TestProductRoundTrip(t *testing.T) {
	p := loadProduct(t, "product_2026-06-08.json")
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var again Product
	if err := json.Unmarshal(data, &again); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p, again) {
		t.Errorf("round trip changed the product:\n%+v\n%+v", p, again)
	}
}

// TestProductMatchesLiveSchema checks the Go types against the Product JSON
// schema served by the API for CompatibilityDate, saved in testdata. It fails
// when the API adds required fields the types do not know about.
func TestProductMatchesLiveSchema(t *testing.T) {
	data, err := os.ReadFile("testdata/schema_product_" + CompatibilityDate + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Schema jsonSchema `json:"schema"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	checkSchema(t, "Product", doc.Schema, reflect.TypeOf(Product{}))
}

type jsonSchema struct {
	Type       string                `json:"type"`
	Required   []string              `json:"required"`
	Properties map[string]jsonSchema `json:"properties"`
	Items      *jsonSchema           `json:"items"`
}

func checkSchema(t *testing.T, path string, s jsonSchema, typ reflect.Type) {
	t.Helper()
	for typ.Kind() == reflect.Slice || typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch s.Type {
	case "array":
		if s.Items != nil {
			checkSchema(t, path+"[]", *s.Items, typ)
		}
		return
	case "object":
	default:
		return
	}

	fields := map[string]reflect.Type{}
	collectJSONFields(typ, fields)
	for _, name := range s.Required {
		ft, ok := fields[name]
		if !ok {
			t.Errorf("%s.%s is required by the API schema but missing from %s", path, name, typ)
			continue
		}
		checkSchema(t, path+"."+name, s.Properties[name], ft)
	}
}

func collectJSONFields(typ reflect.Type, out map[string]reflect.Type) {
	if typ.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Anonymous {
			collectJSONFields(f.Type, out)
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name != "" && name != "-" {
			out[name] = f.Type
		}
	}
}

func TestTimeUnmarshal(t *testing.T) {
	ms := time.UnixMilli(1790867606123).UTC()
	tests := []struct {
		in   string
		want time.Time
	}{
		{`null`, time.Time{}},
		{`0`, time.Time{}},
		{`""`, time.Time{}},
		{`1790867606123`, ms},
		{`1790867606123.0`, ms},
		{`"1790867606123"`, ms},
		{`"2026-10-01T15:13:26.123Z"`, ms},
		{`"2026-10-01T17:13:26.123+02:00"`, ms},
	}
	for _, tt := range tests {
		var got Time
		if err := json.Unmarshal([]byte(tt.in), &got); err != nil {
			t.Errorf("%s: %v", tt.in, err)
			continue
		}
		if !got.Equal(tt.want) {
			t.Errorf("%s: got %v, want %v", tt.in, got.Time, tt.want)
		}
	}

	var bad Time
	if err := json.Unmarshal([]byte(`"soon"`), &bad); err == nil {
		t.Error("expected an error for an invalid timestamp")
	}
}

func TestTimeMarshal(t *testing.T) {
	for _, tt := range []struct {
		in   Time
		want string
	}{
		{Time{}, "0"},
		{Time{time.UnixMilli(1790867606123)}, "1790867606123"},
	} {
		got, err := json.Marshal(tt.in)
		if err != nil || string(got) != tt.want {
			t.Errorf("Marshal(%v) = %s, %v; want %s", tt.in.Time, got, err, tt.want)
		}
	}
}

func TestPriceRoundsFractionalValues(t *testing.T) {
	var p ProductPrice
	if err := json.Unmarshal([]byte(`{"currency":"USD","oldValue":499.6,"newValue":0}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.OldValue != 500 {
		t.Errorf("OldValue = %d, want 500", p.OldValue)
	}
}

func TestDescriptions(t *testing.T) {
	d := Descriptions{
		{Lang: "de-DE", Text: "Deutsch", Priority: 5},
		{Lang: "en-US", Text: "English", Priority: 1},
		{Lang: "fr", Text: "Français", Priority: 9},
	}
	for lang, want := range map[string]string{
		"de-DE": "Deutsch",
		"de":    "Deutsch",
		"DE-at": "Deutsch",
		"fr-CA": "Français",
		"en":    "English",
		"es":    "English", // falls back to English
	} {
		if got := d.Get(lang); got != want {
			t.Errorf("Get(%q) = %q, want %q", lang, got, want)
		}
	}

	noEnglish := Descriptions{{Lang: "de", Text: "a", Priority: 1}, {Lang: "fr", Text: "b", Priority: 3}}
	if got := noEnglish.Default(); got != "b" {
		t.Errorf("Default without English = %q, want highest priority", got)
	}
	if got := (Descriptions{}).Get("en"); got != "" {
		t.Errorf("empty Get = %q", got)
	}

	var fromNull Descriptions
	if err := json.Unmarshal([]byte(`null`), &fromNull); err != nil || fromNull != nil {
		t.Errorf("null: %v %v", fromNull, err)
	}
	var fromEmpty Descriptions
	if err := json.Unmarshal([]byte(`""`), &fromEmpty); err != nil || fromEmpty != nil {
		t.Errorf("empty string: %v %v", fromEmpty, err)
	}
}

func TestProductHelpers(t *testing.T) {
	p := loadProduct(t, "product_2026-06-08.json")

	if v, ok := p.MetaValue(MetaSteamSubIDs); !ok || v != "1,2" {
		t.Errorf("MetaValue = %q, %v", v, ok)
	}
	if _, ok := p.MetaValue(MetaEpicID); ok {
		t.Error("MetaValue found a missing key")
	}

	if pr, ok := p.Price("eur"); !ok || pr.OldValue != 1849 {
		t.Errorf("Price(eur) = %+v, %v", pr, ok)
	}
	if _, ok := p.Price("JPY"); ok {
		t.Error("Price found a missing currency")
	}

	if img, ok := p.BestImage(0); !ok || img.URL != "https://example.com/tags.jpg" {
		t.Errorf("BestImage(0) = %+v", img)
	}
	if img, ok := p.BestImage(ImageFlagLogo); !ok || img.URL != "https://example.com/logo.png" {
		t.Errorf("BestImage(logo) = %+v", img)
	}
	if _, ok := p.BestImage(ImageFlagShowcase); ok {
		t.Error("BestImage found a missing showcase image")
	}

	if u, ok := p.BestURL(0); !ok || u.Priority != 90 {
		t.Errorf("BestURL(0) = %+v", u)
	}
	if u, ok := p.BestURL(URLFlagOriginal); !ok || u.URL != "https://store.example.com/app/1" {
		t.Errorf("BestURL(original) = %+v", u)
	}

	if p.ShouldIgnore() {
		t.Error("ShouldIgnore without the flag")
	}
	p.Flags |= ProductFlagFirstPartyExclusive
	if !p.ShouldIgnore() {
		t.Error("ShouldIgnore with FIRSTPARTY_EXCLUSIVE")
	}
}

func TestFlags(t *testing.T) {
	f := ProductFlagTrash | ProductFlagStaffPick
	if !f.Has(ProductFlagTrash) || !f.Has(ProductFlagTrash|ProductFlagStaffPick) || f.Has(ProductFlagPermanent) || f.Has(0) {
		t.Error("ProductFlags.Has")
	}

	for _, tt := range []struct {
		got, want string
	}{
		{ProductFlags(0).String(), "0"},
		{f.String(), "TRASH|STAFF_PICK"},
		{(ProductFlagPermanent | 1<<12).String(), "PERMANENT|0x1000"},
		{(ImageFlagProxied | ImageFlagWide | ImageFlagPromo).String(), "PROXIED|AR_WIDE|TP_PROMO"},
		{(URLFlagOriginal | URLFlagOpensInBrowser).String(), "ORIGINAL|OPENS_IN_BROWSER"},
	} {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

func TestResolvedAnnouncementDecode(t *testing.T) {
	product, err := os.ReadFile("testdata/product_2026-06-08.json")
	if err != nil {
		t.Fatal(err)
	}
	data := `{"id": 77, "products": [123456], "resolvedProducts": [` + string(product) + `]}`

	var a ResolvedAnnouncement
	if err := json.Unmarshal([]byte(data), &a); err != nil {
		t.Fatal(err)
	}
	if a.ID != 77 || len(a.Products) != 1 || a.Products[0] != 123456 || len(a.ResolvedProducts) != 1 || a.ResolvedProducts[0].Title != "Example Game" {
		t.Errorf("unexpected announcement: %+v", a)
	}
}

func TestProductLenientNumbers(t *testing.T) {
	data := `{"id":1.0e5,"updatedAt":1727793246.5,"flags":-2147483648,
		"images":[{"url":"a","flags":3,"priority":1.5}],
		"urls":[{"url":"b","flags":null,"priority":0.25}],
		"description":[{"lang":"en","text":"x","priority":0.5,"flags":-1}],
		"prices":[{"currency":"USD","oldValue":4.99e2,"newValue":0}]}`
	var p Product
	if err := json.Unmarshal([]byte(data), &p); err != nil {
		t.Fatal(err)
	}
	if p.ID != 100000 || p.UpdatedAt != 1727793247 {
		t.Errorf("ID = %d, UpdatedAt = %d", p.ID, p.UpdatedAt)
	}
	if p.Flags != 1<<31 {
		t.Errorf("Flags = %#x, want bit 31", uint64(p.Flags))
	}
	if p.Images[0].Priority != 1.5 || p.URLs[0].Priority != 0.25 || p.URLs[0].Flags != 0 {
		t.Errorf("images = %+v, urls = %+v", p.Images, p.URLs)
	}
	if p.Description[0].Flags != 0xFFFFFFFF || p.Prices[0].OldValue != 499 {
		t.Errorf("description = %+v, prices = %+v", p.Description, p.Prices)
	}

	for _, bad := range []string{`{"id":"abc"}`, `{"id":1e300}`, `{"flags":1.5e300}`, `{"flags":"x"}`} {
		if err := json.Unmarshal([]byte(bad), &p); err == nil {
			t.Errorf("%s: expected an error", bad)
		}
	}
}

func TestTimeRejectsGarbage(t *testing.T) {
	for _, in := range []string{`"NaN"`, `"Inf"`, `1e300`, `-1e300`} {
		var got Time
		if err := json.Unmarshal([]byte(in), &got); err == nil {
			t.Errorf("%s: expected an error, got %v", in, got.Time)
		}
	}
	var got Time
	if err := json.Unmarshal([]byte(`"2026-10-01T17:13:26+02:00"`), &got); err != nil || got.Location() != time.UTC {
		t.Errorf("RFC 3339 time not normalized to UTC: %v, %v", got.Time, err)
	}
}
