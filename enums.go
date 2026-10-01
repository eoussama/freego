package freego

// The enumerations below are open: FreeStuff documents them as
// non-exhaustive and may add values at any time. Values unknown to this
// version of the library are preserved as-is when decoding.

// Channel is the channel a product was announced in, given as the product's
// Type.
type Channel string

// Known channels.
const (
	// ChannelKeep is "Free to Keep": a temporary 100% discount.
	ChannelKeep Channel = "keep"
	// ChannelTimed is "Free to Play": a free weekend or similar.
	ChannelTimed Channel = "timed"
	// ChannelOther is "DLCs & More": everything else.
	ChannelOther Channel = "other"
	// ChannelPrime is content available through Prime Gaming.
	ChannelPrime Channel = "prime"
	// ChannelGamePass is content available through Game Pass.
	ChannelGamePass Channel = "gamepass"

	// The channels below are not meant for public use.

	ChannelMobile  Channel = "mobile"
	ChannelNews    Channel = "news"
	ChannelUnknown Channel = "unknown"
	ChannelAssets  Channel = "assets"
	ChannelDebug   Channel = "debug"
)

// ProductKind describes what a product is.
type ProductKind string

// Known product kinds.
const (
	KindGame      ProductKind = "game"      // a game
	KindDLC       ProductKind = "dlc"       // a game add-on
	KindLoot      ProductKind = "loot"      // in-game content
	KindSoftware  ProductKind = "software"  // non-game software (unused)
	KindArt       ProductKind = "art"       // digital artwork (unused)
	KindOST       ProductKind = "ost"       // a game soundtrack (unused)
	KindBook      ProductKind = "book"      // a digital book (unused)
	KindStoreItem ProductKind = "storeitem" // e.g. Steam points shop content
	KindOther     ProductKind = "other"     // other gaming-related content
)

// Store is the store a product is offered on. Treat unknown stores as
// [StoreOther].
type Store string

// Known stores.
const (
	StoreOther  Store = "other"
	StoreSteam  Store = "steam"
	StoreEpic   Store = "epic"
	StoreHumble Store = "humble"
	StoreGOG    Store = "gog"
	StoreOrigin Store = "origin"
	StoreUbi    Store = "ubi"
	StoreItch   Store = "itch"
	StorePrime  Store = "prime"
)

// Platform is a platform a product runs on. Ignore unknown platforms.
type Platform string

// Known platforms.
const (
	PlatformWindows     Platform = "windows"
	PlatformMac         Platform = "mac"
	PlatformLinux       Platform = "linux"
	PlatformAndroid     Platform = "android"
	PlatformIOS         Platform = "ios"
	PlatformXbox        Platform = "xbox"
	PlatformPlayStation Platform = "playstation"
)

// Known product meta keys, see [Product.MetaValue]. Keys starting with a
// store name are only present on products from that store.
const (
	MetaSlug                 = "slug"                  // unique id on the product's store
	MetaScraperSources       = "scraper.sources"       // sources used by the scraper
	MetaScraperVersion       = "scraper.version"       // scraper version
	MetaIGDBGameID           = "igdb.gameid"           // id of the game on IGDB
	MetaSteamSubIDs          = "steam.subids"          // comma separated Steam sub ids
	MetaSteamAchievements    = "steam.achievements"    // Steam achievement count
	MetaSteamRecommendations = "steam.recommendations" // Steam curator recommendations
	MetaEpicNamespace        = "epic.namespace"        // Epic Games namespace
	MetaEpicID               = "epic.id"               // Epic Games id
	MetaItchCreator          = "itch.creator"          // itch.io creator name
	MetaPrimeItemID          = "prime.itemid"          // Prime Gaming item id
	MetaPrimeOfferID         = "prime.offerid"         // Prime Gaming offer id
	MetaPrimeCodeGrant       = "prime.codegrant"       // whether the product grants a code
	MetaPrimeDirect          = "prime.direct"          // whether the product is a direct claim
	MetaPrimeLinkItem        = "prime.linkitem"        // whether the product is a retail link item
	MetaPrimeFGWP            = "prime.fgwp"            // whether it is "Free Games With Prime"
	MetaPrimePriority        = "prime.priority"        // Prime Gaming listing priority
)
