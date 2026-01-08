package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/drywaters/seenema/internal/model"
	"github.com/drywaters/seenema/internal/repository"
	"github.com/drywaters/seenema/internal/ui/partials"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RatingHandler handles rating-related requests
type RatingHandler struct {
	ratingRepo *repository.RatingRepository
	entryRepo  *repository.EntryRepository
	personRepo *repository.PersonRepository
}

// NewRatingHandler creates a new RatingHandler
func NewRatingHandler(ratingRepo *repository.RatingRepository, entryRepo *repository.EntryRepository, personRepo *repository.PersonRepository) *RatingHandler {
	return &RatingHandler{
		ratingRepo: ratingRepo,
		entryRepo:  entryRepo,
		personRepo: personRepo,
	}
}

// SaveRating creates or updates a rating
func (h *RatingHandler) SaveRating(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	personIDStr := r.FormValue("person_id")
	entryIDStr := r.FormValue("entry_id")
	scoreStr := r.FormValue("score")

	personID, err := uuid.Parse(personIDStr)
	if err != nil {
		http.Error(w, "Invalid person ID", http.StatusBadRequest)
		return
	}

	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		http.Error(w, "Invalid entry ID", http.StatusBadRequest)
		return
	}

	score, err := strconv.ParseFloat(scoreStr, 64)
	if err != nil || score < 0.0 || score > 10.0 {
		http.Error(w, "Invalid score (must be 0.0-10.0)", http.StatusBadRequest)
		return
	}

	_, err = h.ratingRepo.Upsert(ctx, model.UpsertRatingInput{
		PersonID: personID,
		EntryID:  entryID,
		Score:    score,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		slog.Error("failed to save rating", "error", err)
		http.Error(w, "Failed to save rating", http.StatusInternalServerError)
		return
	}

	// Return updated ratings section
	entry, err := h.entryRepo.GetByID(ctx, entryID)
	if err != nil {
		slog.Error("failed to get entry", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	persons, err := h.personRepo.GetAll(ctx)
	if err != nil {
		slog.Error("failed to get persons", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"showToast": {"message": "Rating saved!", "type": "success"}}`)
	partials.RatingsGrid(entry, persons).Render(ctx, w)
}

// DeleteRating removes a rating
func (h *RatingHandler) DeleteRating(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	personIDStr := chi.URLParam(r, "personId")
	entryIDStr := chi.URLParam(r, "entryId")

	personID, err := uuid.Parse(personIDStr)
	if err != nil {
		http.Error(w, "Invalid person ID", http.StatusBadRequest)
		return
	}

	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		http.Error(w, "Invalid entry ID", http.StatusBadRequest)
		return
	}

	if err := h.ratingRepo.Delete(ctx, personID, entryID); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		slog.Error("failed to delete rating", "error", err)
		http.Error(w, "Failed to delete rating", http.StatusInternalServerError)
		return
	}

	// Return updated ratings section
	entry, err := h.entryRepo.GetByID(ctx, entryID)
	if err != nil {
		slog.Error("failed to get entry", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	persons, err := h.personRepo.GetAll(ctx)
	if err != nil {
		slog.Error("failed to get persons", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"showToast": {"message": "Rating deleted!", "type": "success"}}`)
	partials.RatingsGrid(entry, persons).Render(ctx, w)
}

// RatingForm renders the rating input form for a specific person/entry
func (h *RatingHandler) RatingForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	entryIDStr := chi.URLParam(r, "entryId")
	personIDStr := chi.URLParam(r, "personId")

	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		http.Error(w, "Invalid entry ID", http.StatusBadRequest)
		return
	}

	personID, err := uuid.Parse(personIDStr)
	if err != nil {
		http.Error(w, "Invalid person ID", http.StatusBadRequest)
		return
	}

	entry, err := h.entryRepo.GetByID(ctx, entryID)
	if err != nil {
		slog.Error("failed to get entry", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	person, err := h.personRepo.GetByID(ctx, personID)
	if err != nil {
		slog.Error("failed to get person", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Find existing rating if any
	var existingScore *float64
	for _, rating := range entry.Ratings {
		if rating.PersonID == personID {
			existingScore = &rating.Score
			break
		}
	}

	partials.RatingInputForm(entry.ID, person, existingScore).Render(ctx, w)
}

