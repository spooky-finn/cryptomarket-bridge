package rpc

type ValidationServiceConfig struct {
	AvailableProviders []string
}

type ValidationService struct {
	config *ValidationServiceConfig
}

func NewValidationService(config *ValidationServiceConfig) *ValidationService {
	return &ValidationService{
		config: config,
	}
}

func (s *ValidationService) IsSupportedProvider(provider string) bool {
	for _, p := range s.config.AvailableProviders {
		if p == provider {
			return true
		}
	}
	return false
}
