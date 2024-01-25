package interfaces

type ApiResolver interface {
	ByProvider(provider string) ProviderStreamAPI
	HttpApi(provider string) ProviderHttpAPI
}
