package model

import (
	"time"

	"github.com/google/uuid"
)

// Entry represents a movie entry in a watch group
type Entry struct {
	ID          uuid.UUID  `json:"id"`
	MovieID     uuid.UUID  `json:"movie_id"`
	GroupNumber int        `json:"group_number"`
	WatchedAt   *time.Time `json:"watched_at,omitempty"` // nil = not yet watched
	AddedAt     time.Time  `json:"added_at"`
	Notes       *string    `json:"notes,omitempty"`

	// Joined data (populated by repository)
	Movie   *Movie    `json:"movie,omitempty"`
	Ratings []*Rating `json:"ratings,omitempty"`
}

// CreateEntryInput represents the input for creating an entry
type CreateEntryInput struct {
	MovieID     uuid.UUID `json:"movie_id"`
	GroupNumber int       `json:"group_number"`
	Notes       *string   `json:"notes,omitempty"`
}

// UpdateEntryInput represents the input for updating an entry
type UpdateEntryInput struct {
	GroupNumber *int    `json:"group_number,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

// IsWatched returns true if the entry has been watched
func (e *Entry) IsWatched() bool {
	return e.WatchedAt != nil
}

// AverageRating returns the average rating for this entry, or nil if no ratings
func (e *Entry) AverageRating() *float64 {
	if len(e.Ratings) == 0 {
		return nil
	}

	var sum float64
	for _, r := range e.Ratings {
		sum += r.Score
	}
	avg := sum / float64(len(e.Ratings))
	return &avg
}

// RatingCount returns the number of ratings for this entry
func (e *Entry) RatingCount() int {
	return len(e.Ratings)
}

// IsFullyRated returns true if all family members have rated
func (e *Entry) IsFullyRated() bool {
	return len(e.Ratings) == len(FamilyInitials)
}

// GetRatingByPersonID returns the rating for a specific person, or nil if not rated
func (e *Entry) GetRatingByPersonID(personID uuid.UUID) *Rating {
	for _, r := range e.Ratings {
		if r.PersonID == personID {
			return r
		}
	}
	return nil
}

// GetRatingByInitial returns the rating for a specific person by their initial
func (e *Entry) GetRatingByInitial(initial string) *Rating {
	for _, r := range e.Ratings {
		if r.Person != nil && r.Person.Initial == initial {
			return r
		}
	}
	return nil
}
