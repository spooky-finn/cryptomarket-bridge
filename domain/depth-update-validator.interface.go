package domain

import "errors"

var (
	// This kind of error requesre to be registered and after some thrashoyld reached, the orderbool should be recreated
	ErrOrderBookUpdateIsOutOfSequece = errors.New("order book update is out of sequece")
	// should gust skip them
	ErrOrderBookUpdateIsOutdated = errors.New("order book update is outdated")
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
