package domain

import (
	"fmt"
	"strings"
)

type MarketSymbol struct {
	BaseAsset  string
	QuoteAsset string
}

func NewMarketSymbol(base string, quote string) (*MarketSymbol, error) {
	// check if base and quote are valid
	// i.e. not empty, not equal, etc.

	if base == quote {
		return nil, fmt.Errorf("base and quote must be different")
	}

	if base == "" || quote == "" {
		return nil, fmt.Errorf("base and quote must not be empty")
	}

	// transform base and quote to lowercase
	base = strings.ToLower(base)
	quote = strings.ToLower(quote)

	return &MarketSymbol{
		BaseAsset:  base,
		QuoteAsset: quote,
	}, nil
}

func NewSymbolFromString(s string) (*MarketSymbol, error) {
	// check if s is valid
	// i.e. not empty, contains '/', etc.

	// split s into base and quote
	split := strings.Split(s, "/")
	if len(split) != 2 {
		return nil, fmt.Errorf("invalid symbol string")
	}

	base := split[0]
	quote := split[1]

	return NewMarketSymbol(base, quote)
}

func (ms *MarketSymbol) Join(separator string) string {
	return fmt.Sprintf("%s%s%s", ms.BaseAsset, separator, ms.QuoteAsset)
}

func (ms *MarketSymbol) String() string {
	return fmt.Sprintf("%s/%s", ms.BaseAsset, ms.QuoteAsset)
}

func (ms *MarketSymbol) Equal(other *MarketSymbol) bool {
	return ms.BaseAsset == other.BaseAsset && ms.QuoteAsset == other.QuoteAsset
}
