package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/drywaters/seenema/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MovieRepository handles database operations for movies
type MovieRepository struct {
	pool *pgxpool.Pool
}

// NewMovieRepository creates a new MovieRepository
func NewMovieRepository(pool *pgxpool.Pool) *MovieRepository {
	return &MovieRepository{pool: pool}
}

// Create inserts a new movie into the database
func (r *MovieRepository) Create(ctx context.Context, input model.CreateMovieInput) (*model.Movie, error) {
	var metadataBytes []byte
	if input.MetadataJSON != nil {
		metadataBytes = input.MetadataJSON
	}

	query := `
		INSERT INTO movies (title, release_year, poster_url, synopsis, runtime_minutes, tmdb_id, imdb_id, metadata_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at, title, release_year, poster_url, synopsis, runtime_minutes, tmdb_id, imdb_id, metadata_json`

	movie := &model.Movie{}
	err := r.pool.QueryRow(ctx, query,
		input.Title,
		input.ReleaseYear,
		input.PosterURL,
		input.Synopsis,
		input.RuntimeMinutes,
		input.TMDBId,
		input.IMDBId,
		metadataBytes,
	).Scan(
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
		return nil, fmt.Errorf("create movie: %w", err)
	}

	return movie, nil
}

// GetByID retrieves a movie by its ID
func (r *MovieRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Movie, error) {
	query := `
		SELECT id, created_at, updated_at, title, release_year, poster_url, synopsis, runtime_minutes, tmdb_id, imdb_id, metadata_json
		FROM movies
		WHERE id = $1`

	movie := &model.Movie{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
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
		return nil, fmt.Errorf("get movie by id: %w", err)
	}

	return movie, nil
}

// GetByTMDBId retrieves a movie by its TMDB ID
func (r *MovieRepository) GetByTMDBId(ctx context.Context, tmdbID int) (*model.Movie, error) {
	query := `
		SELECT id, created_at, updated_at, title, release_year, poster_url, synopsis, runtime_minutes, tmdb_id, imdb_id, metadata_json
		FROM movies
		WHERE tmdb_id = $1`

	movie := &model.Movie{}
	err := r.pool.QueryRow(ctx, query, tmdbID).Scan(
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
		return nil, fmt.Errorf("get movie by tmdb id: %w", err)
	}

	return movie, nil
}

// List retrieves all movies ordered by title
func (r *MovieRepository) List(ctx context.Context) ([]*model.Movie, error) {
	query := `
		SELECT id, created_at, updated_at, title, release_year, poster_url, synopsis, runtime_minutes, tmdb_id, imdb_id, metadata_json
		FROM movies
		ORDER BY title`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list movies: %w", err)
	}
	defer rows.Close()

	var movies []*model.Movie
	for rows.Next() {
		movie := &model.Movie{}
		if err := rows.Scan(
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
			return nil, fmt.Errorf("scan movie: %w", err)
		}
		movies = append(movies, movie)
	}

	return movies, nil
}

// Update updates an existing movie
func (r *MovieRepository) Update(ctx context.Context, id uuid.UUID, input model.UpdateMovieInput) (*model.Movie, error) {
	// Build the update query dynamically based on provided fields
	movie, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if movie == nil {
		return nil, nil
	}

	if input.Title != nil {
		movie.Title = *input.Title
	}
	if input.ReleaseYear != nil {
		movie.ReleaseYear = input.ReleaseYear
	}
	if input.PosterURL != nil {
		movie.PosterURL = input.PosterURL
	}
	if input.Synopsis != nil {
		movie.Synopsis = input.Synopsis
	}
	if input.RuntimeMinutes != nil {
		movie.RuntimeMinutes = input.RuntimeMinutes
	}
	if input.IMDBId != nil {
		movie.IMDBId = input.IMDBId
	}
	if input.MetadataJSON != nil {
		movie.MetadataJSON = input.MetadataJSON
	}

	var metadataBytes []byte
	if movie.MetadataJSON != nil {
		metadataBytes, _ = json.Marshal(movie.MetadataJSON)
	}

	query := `
		UPDATE movies
		SET title = $2, release_year = $3, poster_url = $4, synopsis = $5, runtime_minutes = $6, imdb_id = $7, metadata_json = $8
		WHERE id = $1
		RETURNING id, created_at, updated_at, title, release_year, poster_url, synopsis, runtime_minutes, tmdb_id, imdb_id, metadata_json`

	updated := &model.Movie{}
	err = r.pool.QueryRow(ctx, query,
		id,
		movie.Title,
		movie.ReleaseYear,
		movie.PosterURL,
		movie.Synopsis,
		movie.RuntimeMinutes,
		movie.IMDBId,
		metadataBytes,
	).Scan(
		&updated.ID,
		&updated.CreatedAt,
		&updated.UpdatedAt,
		&updated.Title,
		&updated.ReleaseYear,
		&updated.PosterURL,
		&updated.Synopsis,
		&updated.RuntimeMinutes,
		&updated.TMDBId,
		&updated.IMDBId,
		&updated.MetadataJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("update movie: %w", err)
	}

	return updated, nil
}

// Delete removes a movie from the database
func (r *MovieRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM movies WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete movie: %w", err)
	}
	return nil
}

