package ledger

import "errors"

var (
	ErrInsufficientBalance   = errors.New("insufficient balance")
	ErrReservationNotFound   = errors.New("reservation not found")
	ErrReservationNotPending = errors.New("reservation is not in pending state")
)
