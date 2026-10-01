package freego

import (
	"context"
	"encoding/json"
	"net/url"
)

// Schema URNs accepted by [Client.Schema].
const (
	SchemaProduct              = "fsb:schema:apiv2:product"
	SchemaPartialProduct       = "fsb:schema:apiv2:partial_product"
	SchemaAnnouncement         = "fsb:schema:apiv2:announcement"
	SchemaResolvedAnnouncement = "fsb:schema:apiv2:resolved_announcement"
)

// SchemaInfo describes a data model published by the API.
type SchemaInfo struct {
	Name   string `json:"name"`
	URN    string `json:"urn"`
	Active bool   `json:"active"`
	// LatestVersion is the newest compatibility date that changed this
	// model.
	LatestVersion string `json:"latestVersion"`
}

// ProblemInfo describes a problem type the API may return.
type ProblemInfo struct {
	URN string `json:"urn"`
	// Status is FreeStuff's internal status code for the problem. It is
	// usually, but not always, an HTTP status code.
	Status int    `json:"status"`
	Title  string `json:"title"`
}

// EventInfo describes a webhook event type.
type EventInfo struct {
	URN                string `json:"urn"`
	Description        string `json:"description"`
	PayloadDescription string `json:"payloadDescription"`
	// PayloadSchema is the JSON schema (https://json-schema.org/) of the
	// event's data field.
	PayloadSchema json.RawMessage `json:"payloadSchema"`
}

// Ping checks that the API is reachable and that the API key is accepted. It
// is available on every plan.
//
// The /ping endpoint listed in FreeStuff's documentation is not served by the
// API, so Ping requests the lightweight, authenticated schema list instead.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.get(ctx, "/static/schemas", nil, "", nil)
	return err
}

// Schemas lists the data models published by the API. Available on every
// plan.
func (c *Client) Schemas(ctx context.Context) ([]SchemaInfo, error) {
	var body struct {
		List []SchemaInfo `json:"list"`
	}
	if _, err := c.get(ctx, "/static/schemas", nil, "", &body); err != nil {
		return nil, err
	}
	return body.List, nil
}

// Schema returns the JSON schema (https://json-schema.org/) of the data model
// identified by urn (see the Schema* constants) at the client's compatibility
// date. Available on every plan.
func (c *Client) Schema(ctx context.Context, urn string) (json.RawMessage, error) {
	var body struct {
		Schema json.RawMessage `json:"schema"`
	}
	if _, err := c.get(ctx, "/static/schemas/"+url.PathEscape(urn), nil, "", &body); err != nil {
		return nil, err
	}
	return body.Schema, nil
}

// Problems lists the problem types the API may return. Available on every
// plan.
func (c *Client) Problems(ctx context.Context) ([]ProblemInfo, error) {
	var body struct {
		List []ProblemInfo `json:"list"`
	}
	if _, err := c.get(ctx, "/static/problems", nil, "", &body); err != nil {
		return nil, err
	}
	return body.List, nil
}

// Events lists the webhook event types. Available on every plan.
func (c *Client) Events(ctx context.Context) ([]EventInfo, error) {
	var body struct {
		List []EventInfo `json:"list"`
	}
	if _, err := c.get(ctx, "/static/events", nil, "", &body); err != nil {
		return nil, err
	}
	return body.List, nil
}
