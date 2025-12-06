package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrReservationNotFound = errors.New("reservation not found")
	ErrConcurrentUpdate = errors.New("concurrent update detected")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type Balance struct {
	UserID    string
	Balance   int64
	Version   int64
	UpdatedAt time.Time
}

type Reservation struct {
	ID        string
	UserID    string
	AuctionID string
	BidID     string
	Amount    int64
	Status    string // "reserved", "charged", "released"
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (r *Repository) GetBalance(ctx context.Context, userID string) (*Balance, error) {
	var balance Balance
	err := r.db.QueryRow(ctx,
		`SELECT user_id, balance, version, updated_at 
		 FROM balances WHERE user_id = $1`,
		userID,
	).Scan(&balance.UserID, &balance.Balance, &balance.Version, &balance.UpdatedAt)

	if err == pgx.ErrNoRows {
		// Create balance if doesn't exist
		balance = Balance{
			UserID:  userID,
			Balance: 0,
			Version: 0,
		}
		err = r.CreateBalance(ctx, &balance)
		if err != nil {
			return nil, fmt.Errorf("failed to create balance: %w", err)
		}
		return &balance, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return &balance, nil
}

func (r *Repository) CreateBalance(ctx context.Context, balance *Balance) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO balances (user_id, balance, version, updated_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id) DO NOTHING`,
		balance.UserID, balance.Balance, balance.Version, time.Now(),
	)
	return err
}

func (r *Repository) ReserveFunds(ctx context.Context, userID string, amount int64, auctionID, bidID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var balance int64
	var version int64

	err = tx.QueryRow(ctx,
		`SELECT balance, version FROM balances WHERE user_id = $1 FOR UPDATE`,
		userID,
	).Scan(&balance, &version)

	if err == pgx.ErrNoRows {
		// Create balance with 0
		_, err = tx.Exec(ctx,
			`INSERT INTO balances (user_id, balance, version, updated_at)
			 VALUES ($1, 0, 0, $2)`,
			userID, time.Now(),
		)
		if err != nil {
			return "", fmt.Errorf("failed to create balance: %w", err)
		}
		balance = 0
		version = 0
	} else if err != nil {
		return "", fmt.Errorf("failed to get balance: %w", err)
	}

	if balance < amount {
		return "", ErrInsufficientFunds
	}

	reservationID := uuid.New().String()
	_, err = tx.Exec(ctx,
		`INSERT INTO reservations (id, user_id, auction_id, bid_id, amount, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, 'reserved', $6, $6)`,
		reservationID, userID, auctionID, bidID, amount, time.Now(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create reservation: %w", err)
	}

	result, err := tx.Exec(ctx,
		`UPDATE balances 
		 SET balance = balance - $1, version = version + 1, updated_at = $2
		 WHERE user_id = $3 AND version = $4`,
		amount, time.Now(), userID, version,
	)
	if err != nil {
		return "", fmt.Errorf("failed to update balance: %w", err)
	}

	if result.RowsAffected() == 0 {
		return "", ErrConcurrentUpdate
	}

	if err = tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return reservationID, nil
}

func (r *Repository) ChargeFunds(ctx context.Context, reservationID string) (int64, string, error) {
	var reservation Reservation
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, auction_id, bid_id, amount, status
		 FROM reservations WHERE id = $1`,
		reservationID,
	).Scan(&reservation.ID, &reservation.UserID, &reservation.AuctionID,
		&reservation.BidID, &reservation.Amount, &reservation.Status)

	if err == pgx.ErrNoRows {
		return 0, "", ErrReservationNotFound
	}
	if err != nil {
		return 0, "", fmt.Errorf("failed to get reservation: %w", err)
	}

	if reservation.Status != "reserved" {
		return 0, "", fmt.Errorf("reservation is not in reserved status: %s", reservation.Status)
	}

	_, err = r.db.Exec(ctx,
		`UPDATE reservations 
		 SET status = 'charged', updated_at = $1
		 WHERE id = $2 AND status = 'reserved'`,
		time.Now(), reservationID,
	)
	if err != nil {
		return 0, "", fmt.Errorf("failed to charge reservation: %w", err)
	}

	return reservation.Amount, reservation.UserID, nil
}

func (r *Repository) ReleaseFunds(ctx context.Context, reservationID string) error {
	var reservation Reservation
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, amount, status
		 FROM reservations WHERE id = $1`,
		reservationID,
	).Scan(&reservation.ID, &reservation.UserID, &reservation.Amount, &reservation.Status)

	if err == pgx.ErrNoRows {
		return ErrReservationNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to get reservation: %w", err)
	}

	if reservation.Status == "released" {
		return nil // Already released
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE reservations 
		 SET status = 'released', updated_at = $1
		 WHERE id = $2`,
		time.Now(), reservationID,
	)
	if err != nil {
		return fmt.Errorf("failed to update reservation: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE balances 
		 SET balance = balance + $1, updated_at = $2
		 WHERE user_id = $3`,
		reservation.Amount, time.Now(), reservation.UserID,
	)
	if err != nil {
		return fmt.Errorf("failed to restore balance: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetReservation(ctx context.Context, reservationID string) (*Reservation, error) {
	var reservation Reservation
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, auction_id, bid_id, amount, status, created_at, updated_at
		 FROM reservations WHERE id = $1`,
		reservationID,
	).Scan(&reservation.ID, &reservation.UserID, &reservation.AuctionID,
		&reservation.BidID, &reservation.Amount, &reservation.Status,
		&reservation.CreatedAt, &reservation.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, ErrReservationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reservation: %w", err)
	}

	return &reservation, nil
}

