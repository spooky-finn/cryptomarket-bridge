package interfaces

type ApiResolver interface {
	StreamApi(provider string) ProviderStreamAPI
	HttpApi(provider string) ProviderHttpAPI
}
