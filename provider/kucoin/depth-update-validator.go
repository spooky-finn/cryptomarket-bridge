package kucoin

import "github.com/spooky-finn/cryptobridge/domain"

type KucoinDepthUpdateValidator struct {
	OutOfSequeceErrTrashold int
}

func (v *KucoinDepthUpdateValidator) IsValidUpd(update *domain.OrderBookUpdate, orderBookLastUpdId int64) error {
	if update.SequenceEnd <= orderBookLastUpdId {
		return domain.ErrOrderBookUpdateIsOutdated
	}

	// case when update is older than last update
	if update.SequenceStart <= orderBookLastUpdId+1 && update.SequenceEnd >= orderBookLastUpdId {
		return nil
	} else {
		return domain.ErrOrderBookUpdateIsOutOfSequece
	}

}

func (v *KucoinDepthUpdateValidator) IsErrOutOfSequece(err error) bool {
	return err == domain.ErrOrderBookUpdateIsOutOfSequece
}

func (v *KucoinDepthUpdateValidator) IsErrOutdated(err error) bool {
	return err == domain.ErrOrderBookUpdateIsOutdated
}
