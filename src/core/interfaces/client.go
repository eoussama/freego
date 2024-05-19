package interfaces

type IClient interface {
	Ping() bool
	GetGames()
	GetGameDetails()
}
