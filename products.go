package freego

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

// ProductsQuery filters and pages a [Client.Products] request. The zero value
// matches every product, unresolved, starting at the first page.
type ProductsQuery struct {
	// Limit caps the number of products returned. The API may return fewer
	// products than requested even when more exist; use the Count of the
	// result to find out. 0 or less lets the API pick the page size.
	Limit int
	// Offset skips the first products, for pagination. Negative values are
	// treated as 0.
	Offset int
	// Resolve returns full products. When false, only the fields of a partial
	// product are set: ID, Kind, Until, Type, Flags and Store.
	Resolve bool
	// Type only lists products announced in this channel.
	Type Channel
	// Kind only lists products of this kind.
	Kind ProductKind
	// Store only lists products from this store.
	Store Store
	// IfNoneMatch is an ETag from a previous [ProductList]. When the result
	// has not changed since, the request fails with [ErrNotModified].
	IfNoneMatch string
}

func (q *ProductsQuery) values() url.Values {
	v := url.Values{}
	if q == nil {
		return v
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Offset > 0 {
		v.Set("offset", strconv.Itoa(q.Offset))
	}
	if q.Resolve {
		v.Set("resolve", "true")
	}
	if q.Type != "" {
		v.Set("type", string(q.Type))
	}
	if q.Kind != "" {
		v.Set("kind", string(q.Kind))
	}
	if q.Store != "" {
		v.Set("store", string(q.Store))
	}
	return v
}

// ProductList is a page of products.
type ProductList struct {
	// Products is the current page.
	Products []Product
	// Count is the total number of products matching the query, across all
	// pages.
	Count int
	// ETag identifies the full result of the query, regardless of Limit and
	// Offset. Pass it as [ProductsQuery.IfNoneMatch] to skip unchanged
	// results.
	ETag string
}

// Products returns one page of the products currently known to FreeStuff. A
// nil query matches every product, unresolved. The API pages results even
// when no limit is given: compare the number of products with
// [ProductList.Count], or use [Client.ProductsIter] to walk every page.
//
// Requires the "full" tier: on the free tier it fails with
// [ErrUnavailableForFreeTier].
func (c *Client) Products(ctx context.Context, q *ProductsQuery) (*ProductList, error) {
	var ifNoneMatch string
	if q != nil {
		ifNoneMatch = q.IfNoneMatch
	}

	var body struct {
		Products []Product `json:"products"`
		Count    int       `json:"count"`
	}
	resp, err := c.get(ctx, "/products", q.values(), ifNoneMatch, &body)
	if err != nil {
		return nil, err
	}

	return &ProductList{Products: body.Products, Count: body.Count, ETag: resp.etag}, nil
}

// Product returns the full product with the given id. Requires the "full"
// tier. A product that does not exist fails with an error matching
// [ErrNotFound].
func (c *Client) Product(ctx context.Context, id int64) (*Product, error) {
	var raw json.RawMessage
	if _, err := c.get(ctx, "/products/"+strconv.FormatInt(id, 10), nil, "", &raw); err != nil {
		return nil, err
	}
	return decodeProductResponse(raw)
}

// decodeProductResponse decodes the body of GET /products/:id. FreeStuff does
// not document it, so both a bare product and one wrapped in a "product"
// member are accepted.
func decodeProductResponse(raw json.RawMessage) (*Product, error) {
	var wrapped struct {
		Product json.RawMessage `json:"product"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("freego: decoding product: %w", err)
	}
	if len(wrapped.Product) > 0 && !bytes.Equal(wrapped.Product, []byte("null")) {
		raw = wrapped.Product
	}

	var p Product
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("freego: decoding product: %w", err)
	}
	if p.ID == 0 {
		return nil, errors.New("freego: decoding product: response has no product id")
	}
	return &p, nil
}

// ProductIterator walks every page of a products query. Use it like a
// bufio.Scanner:
//
//	it := client.ProductsIter(&freego.ProductsQuery{Resolve: true})
//	for it.Next(ctx) {
//		p := it.Product()
//		// ...
//	}
//	if err := it.Err(); err != nil {
//		// ...
//	}
//
// Pages are fetched by offset, so the iteration is not a snapshot: when
// products are published or expire while iterating, some may be missed. A
// product is never returned twice. Compare [ProductIterator.ETag] with a new
// query's ETag to detect changes.
type ProductIterator struct {
	client  *Client
	query   ProductsQuery
	page    []Product
	index   int
	fetched int
	count   int
	etag    string
	seen    map[int64]struct{}
	started bool
	done    bool
	err     error
}

// ProductsIter returns an iterator over every product matching q, fetching
// pages lazily. q.Offset is the starting offset and q.Limit the page size (0
// lets the API pick). When q.IfNoneMatch is set and nothing changed, Err
// returns [ErrNotModified].
func (c *Client) ProductsIter(q *ProductsQuery) *ProductIterator {
	it := &ProductIterator{client: c, seen: make(map[int64]struct{})}
	if q != nil {
		it.query = *q
	}
	if it.query.Offset < 0 {
		it.query.Offset = 0
	}
	return it
}

// Next advances to the next product, fetching the next page when needed. It
// returns false when there are no more products or an error occurred.
func (it *ProductIterator) Next(ctx context.Context) bool {
	if it.err != nil {
		return false
	}
	if it.index+1 < len(it.page) {
		it.index++
		return true
	}

	for !it.done {
		q := it.query
		if it.started {
			// The ETag covers every page, so it only needs checking once.
			q.IfNoneMatch = ""
		}
		q.Offset = it.query.Offset + it.fetched

		list, err := it.client.Products(ctx, &q)
		if err != nil {
			it.err = err
			it.page = nil
			return false
		}
		if !it.started {
			it.etag = list.ETag
		}
		it.started = true
		it.count = list.Count
		it.fetched += len(list.Products)
		if len(list.Products) == 0 || it.query.Offset+it.fetched >= list.Count {
			it.done = true
		}

		// Skip products already returned, which happens when products
		// expire between two pages and the following ones shift back.
		// A new slice, as callers may keep pointers from Product.
		page := make([]Product, 0, len(list.Products))
		for _, p := range list.Products {
			if _, dup := it.seen[p.ID]; dup {
				continue
			}
			it.seen[p.ID] = struct{}{}
			page = append(page, p)
		}
		it.page = page
		it.index = 0
		if len(it.page) > 0 {
			return true
		}
	}
	it.page = nil
	return false
}

// Product returns the current product. It is only valid after Next returned
// true.
func (it *ProductIterator) Product() *Product {
	if it.index < 0 || it.index >= len(it.page) {
		return nil
	}
	return &it.page[it.index]
}

// Count returns the total number of products matching the query, as
// reported by the last page fetched.
func (it *ProductIterator) Count() int { return it.count }

// ETag returns the ETag of the query's result, available once the first page
// was fetched.
func (it *ProductIterator) ETag() string { return it.etag }

// Err returns the error that stopped the iteration, if any.
func (it *ProductIterator) Err() error { return it.err }
