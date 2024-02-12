package domain

import "errors"

var (
	ErrOrderBookUpdateIsOutOfSequece = errors.New("order book update is out of sequece")
	ErrOrderBookUpdateIsOutdated     = errors.New("order book update is outdated")
)

type DepthUpdateValidator struct {
	OutOfSequeceErrTrashold int
}

type IDepthUpdateValidator interface {
	// if return nil, the update is valid
	IsValidUpd(update *OrderBookUpdate, orderBookLastUpdId int64) error
	IsErrOutOfSequece(err error) bool
	IsErrOutdated(err error) bool
}
