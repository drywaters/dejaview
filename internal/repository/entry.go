package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/drywaters/seenema/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EntryRepository handles database operations for entries
type EntryRepository struct {
	pool *pgxpool.Pool
}

// NewEntryRepository creates a new EntryRepository
func NewEntryRepository(pool *pgxpool.Pool) *EntryRepository {
	return &EntryRepository{pool: pool}
}

// Create inserts a new entry into the database
func (r *EntryRepository) Create(ctx context.Context, input model.CreateEntryInput) (*model.Entry, error) {
	query := `
		INSERT INTO entries (movie_id, group_number, notes)
		VALUES ($1, $2, $3)
		RETURNING id, movie_id, group_number, watched_at, added_at, notes`

	entry := &model.Entry{}
	err := r.pool.QueryRow(ctx, query,
		input.MovieID,
		input.GroupNumber,
		input.Notes,
	).Scan(
		&entry.ID,
		&entry.MovieID,
		&entry.GroupNumber,
		&entry.WatchedAt,
		&entry.AddedAt,
		&entry.Notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create entry: %w", err)
	}

	return entry, nil
}

// GetByID retrieves an entry by its ID with movie and ratings
func (r *EntryRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Entry, error) {
	query := `
		SELECT e.id, e.movie_id, e.group_number, e.watched_at, e.added_at, e.notes,
		       m.id, m.created_at, m.updated_at, m.title, m.release_year, m.poster_url, m.synopsis, m.runtime_minutes, m.tmdb_id, m.imdb_id, m.metadata_json
		FROM entries e
		JOIN movies m ON e.movie_id = m.id
		WHERE e.id = $1`

	entry := &model.Entry{}
	movie := &model.Movie{}

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&entry.ID,
		&entry.MovieID,
		&entry.GroupNumber,
		&entry.WatchedAt,
		&entry.AddedAt,
		&entry.Notes,
		&movie.ID,
		&movie.CreatedAt,
		&movie.UpdatedAt,
		&movie.Title,
		&movie.ReleaseYear,
		&movie.PosterURL,
		&movie.Synopsis,
		&movie.RuntimeMinutes,
		&movie.TMDBId,
		&movie.IMDBId,
		&movie.MetadataJSON,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get entry by id: %w", err)
	}

	entry.Movie = movie

	// Fetch ratings with person info
	ratings, err := r.getRatingsForEntry(ctx, id)
	if err != nil {
		return nil, err
	}
	entry.Ratings = ratings

	return entry, nil
}

// getRatingsForEntry fetches all ratings for an entry with person information
func (r *EntryRepository) getRatingsForEntry(ctx context.Context, entryID uuid.UUID) ([]*model.Rating, error) {
	query := `
		SELECT r.id, r.person_id, r.entry_id, r.score, r.created_at, r.updated_at,
		       p.id, p.initial, p.name
		FROM ratings r
		JOIN persons p ON r.person_id = p.id
		WHERE r.entry_id = $1
		ORDER BY p.initial`

	rows, err := r.pool.Query(ctx, query, entryID)
	if err != nil {
		return nil, fmt.Errorf("get ratings for entry: %w", err)
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

	return ratings, nil
}

// ListByGroup retrieves all entries for a specific group with movie and ratings
func (r *EntryRepository) ListByGroup(ctx context.Context, groupNumber int) ([]*model.Entry, error) {
	query := `
		SELECT e.id, e.movie_id, e.group_number, e.watched_at, e.added_at, e.notes,
		       m.id, m.created_at, m.updated_at, m.title, m.release_year, m.poster_url, m.synopsis, m.runtime_minutes, m.tmdb_id, m.imdb_id, m.metadata_json
		FROM entries e
		JOIN movies m ON e.movie_id = m.id
		WHERE e.group_number = $1
		ORDER BY e.added_at DESC`

	rows, err := r.pool.Query(ctx, query, groupNumber)
	if err != nil {
		return nil, fmt.Errorf("list entries by group: %w", err)
	}
	defer rows.Close()

	var entries []*model.Entry
	for rows.Next() {
		entry := &model.Entry{}
		movie := &model.Movie{}
		if err := rows.Scan(
			&entry.ID,
			&entry.MovieID,
			&entry.GroupNumber,
			&entry.WatchedAt,
			&entry.AddedAt,
			&entry.Notes,
			&movie.ID,
			&movie.CreatedAt,
			&movie.UpdatedAt,
			&movie.Title,
			&movie.ReleaseYear,
			&movie.PosterURL,
			&movie.Synopsis,
			&movie.RuntimeMinutes,
			&movie.TMDBId,
			&movie.IMDBId,
			&movie.MetadataJSON,
		); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entry.Movie = movie
		entries = append(entries, entry)
	}

	// Fetch ratings for all entries
	for _, entry := range entries {
		ratings, err := r.getRatingsForEntry(ctx, entry.ID)
		if err != nil {
			return nil, err
		}
		entry.Ratings = ratings
	}

	return entries, nil
}

// ListGroups returns all unique group numbers in ascending order
func (r *EntryRepository) ListGroups(ctx context.Context) ([]int, error) {
	query := `SELECT DISTINCT group_number FROM entries ORDER BY group_number`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	var groups []int
	for rows.Next() {
		var group int
		if err := rows.Scan(&group); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		groups = append(groups, group)
	}

	return groups, nil
}

// GetCurrentGroup returns the highest group number, or 1 if no entries exist
func (r *EntryRepository) GetCurrentGroup(ctx context.Context) (int, error) {
	query := `SELECT COALESCE(MAX(group_number), 1) FROM entries`

	var group int
	err := r.pool.QueryRow(ctx, query).Scan(&group)
	if err != nil {
		return 1, fmt.Errorf("get current group: %w", err)
	}

	return group, nil
}

// Update updates an existing entry
func (r *EntryRepository) Update(ctx context.Context, id uuid.UUID, input model.UpdateEntryInput) (*model.Entry, error) {
	// Get current entry first
	entry, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	groupNumber := entry.GroupNumber
	notes := entry.Notes

	if input.GroupNumber != nil {
		groupNumber = *input.GroupNumber
	}
	if input.Notes != nil {
		notes = input.Notes
	}

	query := `
		UPDATE entries
		SET group_number = $2, notes = $3
		WHERE id = $1
		RETURNING id, movie_id, group_number, watched_at, added_at, notes`

	updated := &model.Entry{}
	err = r.pool.QueryRow(ctx, query, id, groupNumber, notes).Scan(
		&updated.ID,
		&updated.MovieID,
		&updated.GroupNumber,
		&updated.WatchedAt,
		&updated.AddedAt,
		&updated.Notes,
	)
	if err != nil {
		return nil, fmt.Errorf("update entry: %w", err)
	}

	// Return full entry with movie and ratings
	return r.GetByID(ctx, id)
}

// Delete removes an entry from the database
func (r *EntryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM entries WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete entry: %w", err)
	}
	return nil
}

// SetWatchedDate marks an entry as watched on the given date
func (r *EntryRepository) SetWatchedDate(ctx context.Context, id uuid.UUID, watchedAt time.Time) error {
	query := `UPDATE entries SET watched_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, watchedAt)
	if err != nil {
		return fmt.Errorf("set watched date: %w", err)
	}
	return nil
}

// ClearWatchedDate clears the watched date for an entry
func (r *EntryRepository) ClearWatchedDate(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE entries SET watched_at = NULL WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("clear watched date: %w", err)
	}
	return nil
}

