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

// GetByMovieAndGroup retrieves an entry by movie ID and group number
func (r *EntryRepository) GetByMovieAndGroup(ctx context.Context, movieID uuid.UUID, groupNumber int) (*model.Entry, error) {
	query := `
		SELECT id, movie_id, group_number, watched_at, added_at, notes
		FROM entries
		WHERE movie_id = $1 AND group_number = $2`

	entry := &model.Entry{}
	err := r.pool.QueryRow(ctx, query, movieID, groupNumber).Scan(
		&entry.ID,
		&entry.MovieID,
		&entry.GroupNumber,
		&entry.WatchedAt,
		&entry.AddedAt,
		&entry.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get entry by movie and group: %w", err)
	}

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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ratings rows: %w", err)
	}

	return ratings, nil
}

// getRatingsForEntries fetches all ratings for multiple entries with person information
func (r *EntryRepository) getRatingsForEntries(ctx context.Context, entryIDs []uuid.UUID) (map[uuid.UUID][]*model.Rating, error) {
	ratingsByEntry := make(map[uuid.UUID][]*model.Rating, len(entryIDs))
	if len(entryIDs) == 0 {
		return ratingsByEntry, nil
	}

	query := `
		SELECT r.id, r.person_id, r.entry_id, r.score, r.created_at, r.updated_at,
		       p.id, p.initial, p.name
		FROM ratings r
		JOIN persons p ON r.person_id = p.id
		WHERE r.entry_id = ANY($1)
		ORDER BY r.entry_id, p.initial`

	rows, err := r.pool.Query(ctx, query, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("get ratings for entries: %w", err)
	}
	defer rows.Close()

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
		ratingsByEntry[rating.EntryID] = append(ratingsByEntry[rating.EntryID], rating)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ratings rows: %w", err)
	}

	return ratingsByEntry, nil
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list entries by group rows: %w", err)
	}

	entryIDs := make([]uuid.UUID, 0, len(entries))
	for _, entry := range entries {
		entryIDs = append(entryIDs, entry.ID)
	}

	ratingsByEntry, err := r.getRatingsForEntries(ctx, entryIDs)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		entry.Ratings = ratingsByEntry[entry.ID]
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate groups rows: %w", err)
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
func (r *EntryRepository) Update(ctx context.Context, id uuid.UUID, input model.UpdateEntryInput) error {
	query := `
		UPDATE entries
		SET group_number = COALESCE($2, group_number),
		    notes = COALESCE($3, notes)
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id, input.GroupNumber, input.Notes)
	if err != nil {
		return fmt.Errorf("update entry: %w", err)
	}
	return nil
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
