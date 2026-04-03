package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rmewborne/jellygate/internal/domain"
)

const inviteCols = `id, token, label, created_by, created_at, expires_at, max_uses, use_count, library_ids, revoked`

// InviteStore is a Postgres-backed implementation of domain.InviteStore.
type InviteStore struct {
	pool *pgxpool.Pool
}

// NewInviteStore returns an InviteStore backed by the given pool.
func NewInviteStore(pool *pgxpool.Pool) *InviteStore {
	return &InviteStore{pool: pool}
}

// Create persists a new invite.
func (s *InviteStore) Create(ctx context.Context, inv domain.Invite) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO invites (id, token, label, created_by, expires_at, max_uses, library_ids)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		inv.ID, inv.Token, inv.Label, inv.CreatedBy,
		inv.ExpiresAt, inv.MaxUses, inv.LibraryIDs,
	)
	if err != nil {
		return fmt.Errorf("store.InviteStore.Create: %w", err)
	}
	return nil
}

// GetByToken returns the invite with the given token.
// Returns domain.ErrInviteNotFound if no invite matches.
func (s *InviteStore) GetByToken(ctx context.Context, token string) (domain.Invite, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+inviteCols+` FROM invites WHERE token = $1`, token)
	if err != nil {
		return domain.Invite{}, fmt.Errorf("store.InviteStore.GetByToken: %w", err)
	}
	inv, err := pgx.CollectOneRow(rows, scanInvite)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Invite{}, domain.ErrInviteNotFound
	}
	if err != nil {
		return domain.Invite{}, fmt.Errorf("store.InviteStore.GetByToken: %w", err)
	}
	return inv, nil
}

// List returns all invites ordered by creation time descending.
func (s *InviteStore) List(ctx context.Context) ([]domain.Invite, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+inviteCols+` FROM invites ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("store.InviteStore.List: %w", err)
	}
	invites, err := pgx.CollectRows(rows, scanInvite)
	if err != nil {
		return nil, fmt.Errorf("store.InviteStore.List: %w", err)
	}
	return invites, nil
}

// Revoke marks an invite as revoked.
func (s *InviteStore) Revoke(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE invites SET revoked = TRUE WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store.InviteStore.Revoke: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInviteNotFound
	}
	return nil
}

// IncrementUse atomically increments an invite's use counter.
func (s *InviteStore) IncrementUse(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE invites SET use_count = use_count + 1 WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("store.InviteStore.IncrementUse: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInviteNotFound
	}
	return nil
}

// scanInvite is a pgx.RowToFunc that scans a row into a domain.Invite.
func scanInvite(row pgx.CollectableRow) (domain.Invite, error) {
	var inv domain.Invite
	err := row.Scan(
		&inv.ID,
		&inv.Token,
		&inv.Label,
		&inv.CreatedBy,
		&inv.CreatedAt,
		&inv.ExpiresAt,
		&inv.MaxUses,
		&inv.UseCount,
		&inv.LibraryIDs,
		&inv.Revoked,
	)
	return inv, err
}
