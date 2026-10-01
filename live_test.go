//go:build live

// Live tests call the real FreeStuff API. Run them with:
//
//	FREEGO_FREESTUFF_API_KEY=... go test -tags live -run Live ./...

package freego

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"
)

func liveClient(t *testing.T) *Client {
	t.Helper()
	key := os.Getenv("FREEGO_FREESTUFF_API_KEY")
	if key == "" {
		t.Skip("FREEGO_FREESTUFF_API_KEY is not set")
	}
	c, err := New(key, WithUserAgent("freego-live-tests"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func liveContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestLivePing(t *testing.T) {
	if err := liveClient(t).Ping(liveContext(t)); err != nil {
		t.Fatal(err)
	}
}

func TestLiveInvalidKey(t *testing.T) {
	liveClient(t) // skip without a key, to keep live tests opt-in
	c, _ := New("freego-invalid-key")
	if err := c.Ping(liveContext(t)); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestLiveStatic(t *testing.T) {
	c, ctx := liveClient(t), liveContext(t)

	schemas, err := c.Schemas(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range schemas {
		if s.LatestVersion > CompatibilityDate {
			t.Logf("note: schema %s has a newer version %s than CompatibilityDate %s", s.URN, s.LatestVersion, CompatibilityDate)
		}
	}

	problems, err := c.Problems(ctx)
	if err != nil || len(problems) == 0 {
		t.Fatalf("Problems = %d, %v", len(problems), err)
	}
	events, err := c.Events(ctx)
	if err != nil || len(events) == 0 {
		t.Fatalf("Events = %d, %v", len(events), err)
	}
}

// TestLiveProductSchema compares the live Product schema at
// CompatibilityDate with the copy in testdata, which the offline schema test
// checks the Go types against.
func TestLiveProductSchema(t *testing.T) {
	c, ctx := liveClient(t), liveContext(t)

	live, err := c.Schema(ctx, SchemaProduct)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile("testdata/schema_product_" + CompatibilityDate + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var savedDoc struct {
		Schema any `json:"schema"`
	}
	var liveSchema any
	if err := json.Unmarshal(saved, &savedDoc); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(live, &liveSchema); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(savedDoc.Schema, liveSchema) {
		t.Errorf("the live Product schema for %s differs from testdata; update the testdata and the Go types:\n%s", CompatibilityDate, live)
	}
}

func TestLiveProducts(t *testing.T) {
	c, ctx := liveClient(t), liveContext(t)

	list, err := c.Products(ctx, &ProductsQuery{Limit: 3, Resolve: true})
	if errors.Is(err, ErrUnavailableForFreeTier) {
		t.Skip("the API key is on the free tier, which cannot use content endpoints")
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d products in total, etag %s", list.Count, list.ETag)

	if list.ETag != "" {
		_, err := c.Products(ctx, &ProductsQuery{Limit: 3, Resolve: true, IfNoneMatch: list.ETag})
		if !errors.Is(err, ErrNotModified) {
			t.Logf("note: repeating the query with its ETag returned %v instead of ErrNotModified", err)
		}
	}

	if len(list.Products) == 0 {
		return
	}
	p, err := c.Product(ctx, list.Products[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != list.Products[0].ID || p.Title == "" {
		t.Errorf("unexpected product: %+v", p)
	}
}
