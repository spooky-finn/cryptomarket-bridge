package interfaces

type ApiResolver interface {
	ByProvider(provider string) ProviderStreamAPI
}
