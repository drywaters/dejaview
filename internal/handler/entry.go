package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/drywaters/seenema/internal/model"
	"github.com/drywaters/seenema/internal/repository"
	"github.com/drywaters/seenema/internal/ui/partials"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// EntryHandler handles entry-related requests
type EntryHandler struct {
	entryRepo  *repository.EntryRepository
	personRepo *repository.PersonRepository
}

// NewEntryHandler creates a new EntryHandler
func NewEntryHandler(entryRepo *repository.EntryRepository, personRepo *repository.PersonRepository) *EntryHandler {
	return &EntryHandler{
		entryRepo:  entryRepo,
		personRepo: personRepo,
	}
}

// Update updates an entry
func (h *EntryHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	entryIDStr := chi.URLParam(r, "id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		http.Error(w, "Invalid entry ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	input := model.UpdateEntryInput{}

	if groupStr := r.FormValue("group_number"); groupStr != "" {
		groupNumber, err := strconv.Atoi(groupStr)
		if err == nil {
			input.GroupNumber = &groupNumber
		}
	}

	if notes := r.FormValue("notes"); notes != "" {
		input.Notes = &notes
	}

	if _, ok := r.Form["picked_by_person_id"]; ok {
		pickedByStr := r.FormValue("picked_by_person_id")
		if pickedByStr == "" {
			nilID := uuid.Nil
			input.PickedByPersonID = &nilID
		} else {
			pickedByID, err := uuid.Parse(pickedByStr)
			if err != nil {
				slog.Warn("invalid picked_by_person_id", "error", err, "picked_by_person_id", pickedByStr, "entry_id", entryID)
				http.Error(w, "Invalid picked_by_person_id", http.StatusBadRequest)
				return
			}
			input.PickedByPersonID = &pickedByID
		}
	}

	err = h.entryRepo.Update(ctx, entryID, input)
	if err != nil {
		slog.Error("failed to update entry", "error", err)
		http.Error(w, "Failed to update entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"showToast": {"message": "Entry updated!", "type": "success"}, "refreshGroups": true}`)
	w.WriteHeader(http.StatusOK)
}

// Delete removes an entry
func (h *EntryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	entryIDStr := chi.URLParam(r, "id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		http.Error(w, "Invalid entry ID", http.StatusBadRequest)
		return
	}

	if err := h.entryRepo.Delete(ctx, entryID); err != nil {
		slog.Error("failed to delete entry", "error", err)
		http.Error(w, "Failed to delete entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"showToast": {"message": "Entry deleted!", "type": "success"}, "refreshGroups": true}`)
	w.WriteHeader(http.StatusOK)
}

// MarkWatched marks an entry as watched
func (h *EntryHandler) MarkWatched(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	entryIDStr := chi.URLParam(r, "id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		http.Error(w, "Invalid entry ID", http.StatusBadRequest)
		return
	}

	// Use today's date as the watched date
	watchedAt := time.Now()

	// Check if a specific date was provided
	if dateStr := r.FormValue("watched_at"); dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err == nil {
			watchedAt = parsedDate
		}
	}

	if err := h.entryRepo.SetWatchedDate(ctx, entryID, watchedAt); err != nil {
		slog.Error("failed to mark watched", "error", err)
		http.Error(w, "Failed to mark as watched", http.StatusInternalServerError)
		return
	}

	// Return updated entry partial
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

	w.Header().Set("HX-Trigger", `{"showToast": {"message": "Marked as watched!", "type": "success"}}`)
	partials.WatchedStatus(entry, persons).Render(ctx, w)
}

// ClearWatched clears the watched date
func (h *EntryHandler) ClearWatched(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	entryIDStr := chi.URLParam(r, "id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		http.Error(w, "Invalid entry ID", http.StatusBadRequest)
		return
	}

	if err := h.entryRepo.ClearWatchedDate(ctx, entryID); err != nil {
		slog.Error("failed to clear watched", "error", err)
		http.Error(w, "Failed to clear watched status", http.StatusInternalServerError)
		return
	}

	// Return updated entry partial
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

	w.Header().Set("HX-Trigger", `{"showToast": {"message": "Cleared watched status!", "type": "success"}}`)
	partials.WatchedStatus(entry, persons).Render(ctx, w)
}

// GroupPartial renders a single group section
func (h *EntryHandler) GroupPartial(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	groupNumStr := chi.URLParam(r, "num")
	groupNum, err := strconv.Atoi(groupNumStr)
	if err != nil {
		http.Error(w, "Invalid group number", http.StatusBadRequest)
		return
	}

	entries, err := h.entryRepo.ListByGroup(ctx, groupNum)
	if err != nil {
		slog.Error("failed to list entries", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	persons, err := h.personRepo.GetAll(ctx)
	if err != nil {
		slog.Error("failed to get persons", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	partials.GroupSection(groupNum, entries, persons).Render(ctx, w)
}

// ReorderRequest represents the JSON body for reordering entries
type ReorderRequest struct {
	EntryIDs []string `json:"entry_ids"`
}

// Reorder updates the order of entries within a group
func (h *EntryHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	groupNumStr := chi.URLParam(r, "num")
	groupNum, err := strconv.Atoi(groupNumStr)
	if err != nil {
		http.Error(w, "Invalid group number", http.StatusBadRequest)
		return
	}

	var req ReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Convert string IDs to UUIDs
	entryIDs := make([]uuid.UUID, 0, len(req.EntryIDs))
	for _, idStr := range req.EntryIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			slog.Warn("invalid entry id in reorder request", "entry_id", idStr, "error", err)
			http.Error(w, "Invalid entry ID", http.StatusBadRequest)
			return
		}
		entryIDs = append(entryIDs, id)
	}

	if err := h.entryRepo.ReorderEntries(ctx, groupNum, entryIDs); err != nil {
		slog.Error("failed to reorder entries", "error", err)
		http.Error(w, "Failed to reorder entries", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
