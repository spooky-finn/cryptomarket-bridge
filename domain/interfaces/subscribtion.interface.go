package interfaces

type Subscription[T any] struct {
	Stream      chan T
	Unsubscribe func()
	Topic       string
}
