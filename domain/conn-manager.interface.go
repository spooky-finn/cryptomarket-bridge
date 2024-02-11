package domain

type ConnManager interface {
	StreamAPI(provider string) ProviderStreamAPI
	SyncAPI(provider string) ProviderSyncAPI
}
