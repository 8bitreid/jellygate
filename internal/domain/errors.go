package domain

import "errors"

var (
	ErrInviteNotFound  = errors.New("invite not found")
	ErrInviteRevoked   = errors.New("invite has been revoked")
	ErrInviteExpired   = errors.New("invite has expired")
	ErrInviteExhausted = errors.New("invite has reached its maximum uses")
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session has expired")
)
