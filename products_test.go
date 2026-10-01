package freego

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
)

func TestProductsQueryEncoding(t *testing.T) {
	var query url.Values
	var ifNoneMatch string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query, ifNoneMatch = r.URL.Query(), r.Header.Get("If-None-Match")
		writeJSON(w, 200, map[string]any{"products": []any{}, "count": 0})
	})

	_, err := c.Products(context.Background(), &ProductsQuery{
		Limit: 10, Offset: 20, Resolve: true,
		Type: ChannelKeep, Kind: KindGame, Store: StoreEpic,
		IfNoneMatch: `"abc"`,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := url.Values{"limit": {"10"}, "offset": {"20"}, "resolve": {"true"}, "type": {"keep"}, "kind": {"game"}, "store": {"epic"}}
	if query.Encode() != want.Encode() || ifNoneMatch != `"abc"` {
		t.Errorf("query = %v, If-None-Match = %q", query, ifNoneMatch)
	}

	if _, err := c.Products(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if len(query) != 0 || ifNoneMatch != "" {
		t.Errorf("nil query sent %v, %q", query, ifNoneMatch)
	}
}

func TestProducts(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("ETag", `W/"etag-1"`)
		writeJSON(w, 200, map[string]any{
			"products": []any{map[string]any{"id": 123456, "kind": "game", "until": 0, "type": "keep", "flags": 0, "store": "steam"}},
			"count":    1,
		})
	})

	list, err := c.Products(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if list.Count != 1 || list.ETag != `W/"etag-1"` || len(list.Products) != 1 {
		t.Fatalf("unexpected list: %+v", list)
	}
	if p := list.Products[0]; p.ID != 123456 || p.Kind != KindGame || p.Type != ChannelKeep || p.Store != StoreSteam || !p.Until.IsZero() {
		t.Errorf("unexpected partial product: %+v", p)
	}
}

func TestProductsNotModified(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == `W/"same"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		writeJSON(w, 200, map[string]any{"products": []any{}, "count": 0})
	})

	_, err := c.Products(context.Background(), &ProductsQuery{IfNoneMatch: `W/"same"`})
	if !errors.Is(err, ErrNotModified) {
		t.Errorf("err = %v, want ErrNotModified", err)
	}
}

func TestProductsFreeTier(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 403, map[string]any{
			"type":   ProblemTypeUnavailableForFreeTier,
			"title":  "The endpoint you are trying to call is not available for your plan",
			"detail": "Please upgrade to a paid plan to access this endpoint.",
		})
	})
	_, err := c.Products(context.Background(), nil)
	if !errors.Is(err, ErrUnavailableForFreeTier) {
		t.Errorf("err = %v, want ErrUnavailableForFreeTier", err)
	}
}

func TestProduct(t *testing.T) {
	product, err := os.ReadFile("testdata/product_2026-06-08.json")
	if err != nil {
		t.Fatal(err)
	}

	for name, body := range map[string]string{
		"bare":    string(product),
		"wrapped": `{"product":` + string(product) + `}`,
	} {
		t.Run(name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/products/123456" {
					t.Errorf("path = %s", r.URL.Path)
				}
				_, _ = w.Write([]byte(body))
			})
			p, err := c.Product(context.Background(), 123456)
			if err != nil {
				t.Fatal(err)
			}
			if p.ID != 123456 || p.Title != "Example Game" {
				t.Errorf("unexpected product: %+v", p)
			}
		})
	}

	t.Run("no product", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"TODO":true}`))
		})
		if _, err := c.Product(context.Background(), 1); err == nil {
			t.Error("expected an error for a body without a product")
		}
	})
}

// pagedServer serves total products in pages of at most pageSize, ignoring
// the requested limit to mimic the API's own paging.
func pagedServer(total, pageSize int, requests *atomic.Int32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		var products []any
		for i := offset; i < total && i < offset+pageSize; i++ {
			products = append(products, map[string]any{"id": i + 1, "kind": "game", "type": "keep", "store": "steam", "until": 0, "flags": 0})
		}
		if products == nil {
			products = []any{}
		}
		w.Header().Set("ETag", `W/"all"`)
		writeJSON(w, 200, map[string]any{"products": products, "count": total})
	}
}

func TestProductsIter(t *testing.T) {
	var requests atomic.Int32
	c := newTestClient(t, pagedServer(7, 3, &requests))

	it := c.ProductsIter(nil)
	var ids []int64
	for it.Next(context.Background()) {
		ids = append(ids, it.Product().ID)
	}
	if err := it.Err(); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 7 || ids[0] != 1 || ids[6] != 7 {
		t.Errorf("ids = %v", ids)
	}
	if requests.Load() != 3 {
		t.Errorf("requests = %d, want 3", requests.Load())
	}
	if it.Count() != 7 || it.ETag() != `W/"all"` {
		t.Errorf("Count = %d, ETag = %q", it.Count(), it.ETag())
	}
	if it.Next(context.Background()) {
		t.Error("Next after the end returned true")
	}
}

func TestProductsIterOffsetAndEmpty(t *testing.T) {
	var requests atomic.Int32
	c := newTestClient(t, pagedServer(5, 10, &requests))

	it := c.ProductsIter(&ProductsQuery{Offset: 3})
	n := 0
	for it.Next(context.Background()) {
		n++
	}
	if n != 2 || it.Err() != nil || requests.Load() != 1 {
		t.Errorf("n = %d, err = %v, requests = %d", n, it.Err(), requests.Load())
	}

	empty := newTestClient(t, pagedServer(0, 10, &requests))
	it = empty.ProductsIter(nil)
	if it.Next(context.Background()) || it.Err() != nil || it.Product() != nil {
		t.Error("empty iteration")
	}
}

func TestProductsIterError(t *testing.T) {
	var requests atomic.Int32
	page := pagedServer(10, 4, &requests)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("offset") == "4" {
			writeJSON(w, 502, map[string]any{"type": ProblemTypeBadGateway, "title": "Internal gateway error"})
			return
		}
		page(w, r)
	})

	it := c.ProductsIter(nil)
	n := 0
	for it.Next(context.Background()) {
		n++
	}
	var p *Problem
	if n != 4 || !errors.As(it.Err(), &p) || p.Status != 502 {
		t.Errorf("n = %d, err = %v", n, it.Err())
	}
}

func TestProductsIterNotModified(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	})
	it := c.ProductsIter(&ProductsQuery{IfNoneMatch: `W/"x"`})
	if it.Next(context.Background()) || !errors.Is(it.Err(), ErrNotModified) {
		t.Errorf("err = %v, want ErrNotModified", it.Err())
	}
}

// TestProductsIterShiftedPages simulates product 1 expiring after the first
// page: the following products shift back by one, so the second page starts
// with a product that was already returned.
func TestProductsIterShiftedPages(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		ids := map[int][]int{0: {1, 2}, 2: {2, 4}, 4: {}}[offset]
		products := []any{}
		for _, id := range ids {
			products = append(products, map[string]any{"id": id})
		}
		writeJSON(w, 200, map[string]any{"products": products, "count": 5})
	})

	it := c.ProductsIter(&ProductsQuery{Limit: 2})
	var ids []int64
	var kept []*Product
	for it.Next(context.Background()) {
		ids = append(ids, it.Product().ID)
		kept = append(kept, it.Product())
	}
	if it.Err() != nil || len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 4 {
		t.Errorf("ids = %v, err = %v", ids, it.Err())
	}
	// Pointers returned earlier stay valid.
	if kept[0].ID != 1 || kept[1].ID != 2 {
		t.Errorf("earlier products were overwritten: %d %d", kept[0].ID, kept[1].ID)
	}
}

func TestProductsIterNegativeOffset(t *testing.T) {
	var requests atomic.Int32
	c := newTestClient(t, pagedServer(20, 10, &requests))
	it := c.ProductsIter(&ProductsQuery{Offset: -5})
	var ids []int64
	for it.Next(context.Background()) {
		ids = append(ids, it.Product().ID)
	}
	if len(ids) != 20 || ids[0] != 1 || ids[19] != 20 {
		t.Errorf("ids = %v", ids)
	}
}
