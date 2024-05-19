package models

import (
	"fmt"

	"github.com/eoussama/freego/core/types"
)

type GameInfo struct {
	Url         string                 `json:"url"`
	Tags        []string               `json:"tags"`
	Type        types.AnnouncementType `json:"type"`
	Kind        types.ProductKind      `json:"kind"`
	Store       types.Store            `json:"store"`
	Flags       types.GameFlag         `json:"flags"`
	Title       string                 `json:"title"`
	OrgUrl      string                 `json:"org_url"`
	Description string                 `json:"description"`
	Until       int                    `json:"until,omitempty"`
	Rating      int8                   `json:"rating,omitempty"`
	Notice      string                 `json:"notice,omitempty"`

	StoreMeta struct {
		SteamSubIDs string `json:"steam_subids"`
	} `json:"store_meta"`

	Localized map[string]struct {
		Free          string   `json:"free"`
		Until         string   `json:"until"`
		Flags         []string `json:"flags"`
		Header        string   `json:"header"`
		Footer        string   `json:"string"`
		Platform      string   `json:"platform"`
		LangName      string   `json:"lang_name"`
		UntilAlt      string   `json:"until_alt"`
		ClaimLong     string   `json:"claim_long"`
		ClaimShort    string   `json:"claim_short"`
		LangNameEN    string   `json:"lang_name_en"`
		OrgPriceEur   string   `json:"org_price_eur"`
		OrgPriceUSD   string   `json:"org_price_usd"`
		LangFlagEmoji string   `json:"lang_flag_emoji"`
	} `json:"localized,omitempty"`

	Urls struct {
		Org     string `json:"org"`
		Default string `json:"default"`
		Browser string `json:"browser"`
		Client  string `json:"client,omitempty"`
	} `json:"urls"`

	Thumbnail struct {
		Org   string `json:"org"`
		Blank string `json:"blank"`
		Full  string `json:"full"`
		Tags  string `json:"tags"`
	} `json:"thumbnail"`

	Price struct {
		Euro   int `json:"euro"`
		Dollar int `json:"dollar"`
	} `json:"price"`

	OrgPrice struct {
		Euro   int `json:"euro"`
		Dollar int `json:"dollar"`
	} `json:"org_price"`
}

func (gi GameInfo) From(data map[string]interface{}) GameInfo {
	fmt.Println("data =>", data)
	return GameInfo{
		Title: data["title"].(string),
	}
}
