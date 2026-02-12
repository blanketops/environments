package domain

import "errors"

var (
	ErrMissingProvider   = errors.New("missing provider")
	ErrInvalidRepository = errors.New("invalid repository identity")
	ErrInvalidWebhook    = errors.New("invalid webhook specification")
)
