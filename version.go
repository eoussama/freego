package freego

// Version is the version of this library. It is sent to FreeStuff in the
// User-Agent of REST requests and in the X-Client-Library header of webhook
// responses.
const Version = "0.1.0"

// CompatibilityDate is the FreeStuff API compatibility date this version of
// the library is built against. It selects the shape of the data returned by
// the REST API and delivered over webhooks.
//
// See https://docs.freestuffbot.xyz/api-v2/concepts#compatibility-dates.
const CompatibilityDate = "2026-06-08"

// DefaultBaseURL is the base URL of the FreeStuff REST API v2.
const DefaultBaseURL = "https://api.freestuffbot.xyz/v2"

// ClientLibrary identifies this library in the format FreeStuff asks for:
// "Name/version (homepage)".
const ClientLibrary = "freego/" + Version + " (https://github.com/eoussama/freego)"
