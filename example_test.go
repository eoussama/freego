package freego_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/eoussama/freego"
)

func ExampleNew() {
	client, err := freego.New(os.Getenv("FREEGO_FREESTUFF_API_KEY"),
		freego.WithUserAgent("my-bot/1.0 (https://example.com)"),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		log.Fatal(err)
	}
}

func ExampleClient_Products() {
	client, err := freego.New(os.Getenv("FREEGO_FREESTUFF_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	list, err := client.Products(context.Background(), &freego.ProductsQuery{
		Type:    freego.ChannelKeep,
		Resolve: true,
	})
	switch {
	case errors.Is(err, freego.ErrUnavailableForFreeTier):
		log.Fatal("listing products requires the full tier; use webhooks on the free tier")
	case err != nil:
		log.Fatal(err)
	}

	for _, p := range list.Products {
		if p.ShouldIgnore() {
			continue
		}
		url, _ := p.BestURL(freego.URLFlagOpensInBrowser)
		fmt.Printf("%s on %s until %s: %s\n", p.Title, p.Store, p.Until.Format(time.DateOnly), url.URL)
	}
}

// Poll for changes cheaply by sending back the ETag of the previous result.
func ExampleClient_Products_etag() {
	client, err := freego.New(os.Getenv("FREEGO_FREESTUFF_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	var etag string
	for range time.Tick(10 * time.Minute) {
		list, err := client.Products(context.Background(), &freego.ProductsQuery{IfNoneMatch: etag})
		if errors.Is(err, freego.ErrNotModified) {
			continue
		}
		if err != nil {
			log.Print(err)
			continue
		}
		etag = list.ETag
		fmt.Println(list.Count, "products")
	}
}

func ExampleClient_ProductsIter() {
	client, err := freego.New(os.Getenv("FREEGO_FREESTUFF_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	it := client.ProductsIter(&freego.ProductsQuery{Resolve: true, Limit: 50})
	for it.Next(ctx) {
		p := it.Product()
		fmt.Println(p.ID, p.Title)
	}
	if err := it.Err(); err != nil {
		log.Fatal(err)
	}
}

func ExampleProblem() {
	err := error(&freego.Problem{
		Type:   freego.ProblemTypeUnavailableForFreeTier,
		Title:  "The endpoint you are trying to call is not available for your plan",
		Status: 403,
	})

	var problem *freego.Problem
	if errors.As(err, &problem) {
		fmt.Println(problem.Status, problem.Type)
	}
	fmt.Println(errors.Is(err, freego.ErrUnavailableForFreeTier))
	// Output:
	// 403 fsb:problem:api:unavailable_for_free_tier
	// true
}

func ExampleDescriptions_Get() {
	d := freego.Descriptions{
		{Lang: "en", Text: "An example game."},
		{Lang: "de", Text: "Ein Beispielspiel."},
	}
	fmt.Println(d.Get("de-AT"))
	fmt.Println(d.Get("ja"))
	// Output:
	// Ein Beispielspiel.
	// An example game.
}
