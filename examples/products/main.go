// Command products lists the API's data models and, on the full tier, the
// products currently free on FreeStuff.
//
// Usage:
//
//	FREEGO_FREESTUFF_API_KEY=... go run ./examples/products
//
// FREEGO_FREESTUFF_API_URL optionally overrides the API base URL.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/eoussama/freego"
)

func main() {
	var opts []freego.Option
	if u := os.Getenv("FREEGO_FREESTUFF_API_URL"); u != "" {
		opts = append(opts, freego.WithBaseURL(u))
	}
	client, err := freego.New(os.Getenv("FREEGO_FREESTUFF_API_KEY"), opts...)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		log.Fatalf("ping: %v", err)
	}
	fmt.Println("API reachable, key accepted")

	schemas, err := client.Schemas(ctx)
	if err != nil {
		log.Fatalf("schemas: %v", err)
	}
	fmt.Printf("\nData models (this library uses compatibility date %s):\n", freego.CompatibilityDate)
	for _, s := range schemas {
		fmt.Printf("  %-22s latest version %s\n", s.Name, s.LatestVersion)
	}

	fmt.Println("\nFree to keep right now:")
	it := client.ProductsIter(&freego.ProductsQuery{Type: freego.ChannelKeep, Resolve: true})
	for it.Next(ctx) {
		p := it.Product()
		if p.ShouldIgnore() {
			continue
		}
		until := "no end date"
		if !p.Until.IsZero() {
			until = "until " + p.Until.Local().Format("Mon Jan 2 15:04")
		}
		url, _ := p.BestURL(freego.URLFlagOpensInBrowser)
		fmt.Printf("  %s (%s, %s) %s\n", p.Title, p.Store, until, url.URL)
	}
	switch err := it.Err(); {
	case errors.Is(err, freego.ErrUnavailableForFreeTier):
		fmt.Println("  Listing products requires FreeStuff's full tier. On the free tier, receive them with webhooks (see examples/webhook).")
	case err != nil:
		log.Fatalf("products: %v", err)
	}
}
