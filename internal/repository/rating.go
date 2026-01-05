package repository

import (
	"context"
	"fmt"

	"github.com/drywaters/seenema/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RatingRepository handles database operations for ratings
type RatingRepository struct {
	pool *pgxpool.Pool
}

// NewRatingRepository creates a new RatingRepository
func NewRatingRepository(pool *pgxpool.Pool) *RatingRepository {
	return &RatingRepository{pool: pool}
}

// Upsert creates or updates a rating
func (r *RatingRepository) Upsert(ctx context.Context, input model.UpsertRatingInput) (*model.Rating, error) {
	query := `
		INSERT INTO ratings (person_id, entry_id, score)
		VALUES ($1, $2, $3)
		ON CONFLICT (person_id, entry_id)
		DO UPDATE SET score = $3, updated_at = NOW()
		RETURNING id, person_id, entry_id, score, created_at, updated_at`

	rating := &model.Rating{}
	err := r.pool.QueryRow(ctx, query,
		input.PersonID,
		input.EntryID,
		input.Score,
	).Scan(
		&rating.ID,
		&rating.PersonID,
		&rating.EntryID,
		&rating.Score,
		&rating.CreatedAt,
		&rating.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert rating: %w", err)
	}

	return rating, nil
}

// GetByEntryID retrieves all ratings for an entry with person information
func (r *RatingRepository) GetByEntryID(ctx context.Context, entryID uuid.UUID) ([]*model.Rating, error) {
	query := `
		SELECT r.id, r.person_id, r.entry_id, r.score, r.created_at, r.updated_at,
		       p.id, p.initial, p.name
		FROM ratings r
		JOIN persons p ON r.person_id = p.id
		WHERE r.entry_id = $1
		ORDER BY p.initial`

	rows, err := r.pool.Query(ctx, query, entryID)
	if err != nil {
		return nil, fmt.Errorf("get ratings by entry id: %w", err)
	}
	defer rows.Close()

	var ratings []*model.Rating
	for rows.Next() {
		rating := &model.Rating{}
		person := &model.Person{}
		if err := rows.Scan(
			&rating.ID,
			&rating.PersonID,
			&rating.EntryID,
			&rating.Score,
			&rating.CreatedAt,
			&rating.UpdatedAt,
			&person.ID,
			&person.Initial,
			&person.Name,
		); err != nil {
			return nil, fmt.Errorf("scan rating: %w", err)
		}
		rating.Person = person
		ratings = append(ratings, rating)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ratings rows: %w", err)
	}

	return ratings, nil
}

// Delete removes a rating
func (r *RatingRepository) Delete(ctx context.Context, personID, entryID uuid.UUID) error {
	query := `DELETE FROM ratings WHERE person_id = $1 AND entry_id = $2`
	_, err := r.pool.Exec(ctx, query, personID, entryID)
	if err != nil {
		return fmt.Errorf("delete rating: %w", err)
	}
	return nil
}

// GetAverageForEntry calculates the average rating for an entry
func (r *RatingRepository) GetAverageForEntry(ctx context.Context, entryID uuid.UUID) (*float64, error) {
	query := `SELECT AVG(score)::numeric(3,1) FROM ratings WHERE entry_id = $1`

	var avg *float64
	err := r.pool.QueryRow(ctx, query, entryID).Scan(&avg)
	if err != nil {
		return nil, fmt.Errorf("get average for entry: %w", err)
	}

	return avg, nil
}
