package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Movie represents a movie in the library
type Movie struct {
	ID             uuid.UUID       `json:"id"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Title          string          `json:"title"`
	ReleaseYear    *int            `json:"release_year,omitempty"`
	PosterURL      *string         `json:"poster_url,omitempty"`
	Synopsis       *string         `json:"synopsis,omitempty"`
	RuntimeMinutes *int            `json:"runtime_minutes,omitempty"`
	TMDBId         *int            `json:"tmdb_id,omitempty"`
	IMDBId         *string         `json:"imdb_id,omitempty"`
	MetadataJSON   json.RawMessage `json:"metadata_json,omitempty"`
}

// CreateMovieInput represents the input for creating a movie
type CreateMovieInput struct {
	Title          string          `json:"title"`
	ReleaseYear    *int            `json:"release_year,omitempty"`
	PosterURL      *string         `json:"poster_url,omitempty"`
	Synopsis       *string         `json:"synopsis,omitempty"`
	RuntimeMinutes *int            `json:"runtime_minutes,omitempty"`
	TMDBId         *int            `json:"tmdb_id,omitempty"`
	IMDBId         *string         `json:"imdb_id,omitempty"`
	MetadataJSON   json.RawMessage `json:"metadata_json,omitempty"`
}

// UpdateMovieInput represents the input for updating a movie
type UpdateMovieInput struct {
	Title          *string         `json:"title,omitempty"`
	ReleaseYear    *int            `json:"release_year,omitempty"`
	PosterURL      *string         `json:"poster_url,omitempty"`
	Synopsis       *string         `json:"synopsis,omitempty"`
	RuntimeMinutes *int            `json:"runtime_minutes,omitempty"`
	IMDBId         *string         `json:"imdb_id,omitempty"`
	MetadataJSON   json.RawMessage `json:"metadata_json,omitempty"`
}

// FormattedRuntime returns a human-readable runtime string
func (m *Movie) FormattedRuntime() string {
	if m.RuntimeMinutes == nil {
		return ""
	}
	hours := *m.RuntimeMinutes / 60
	minutes := *m.RuntimeMinutes % 60
	if hours > 0 {
		return formatDuration(hours, minutes)
	}
	return formatMinutes(minutes)
}

func formatDuration(hours, minutes int) string {
	if minutes > 0 {
		return intToStr(hours) + "h " + intToStr(minutes) + "m"
	}
	return intToStr(hours) + "h"
}

func formatMinutes(minutes int) string {
	return intToStr(minutes) + "m"
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

