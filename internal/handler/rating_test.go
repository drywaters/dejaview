package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/drywaters/seenema/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type stubRatingRepo struct {
	deleteCalls int
	upsertCalls int
}

func (s *stubRatingRepo) Upsert(ctx context.Context, input model.UpsertRatingInput) (*model.Rating, error) {
	s.upsertCalls++
	return &model.Rating{
		PersonID: input.PersonID,
		EntryID:  input.EntryID,
		Score:    input.Score,
	}, nil
}

func (s *stubRatingRepo) Delete(ctx context.Context, personID, entryID uuid.UUID) error {
	s.deleteCalls++
	return nil
}

type stubEntryRepo struct {
	entries []*model.Entry
	errs    []error
	calls   int
}

func (s *stubEntryRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Entry, error) {
	defer func() { s.calls++ }()
	if s.calls < len(s.entries) {
		var err error
		if s.calls < len(s.errs) {
			err = s.errs[s.calls]
		}
		return s.entries[s.calls], err
	}
	return nil, nil
}

type stubPersonRepo struct {
	calls int
}

func (s *stubPersonRepo) GetAll(ctx context.Context) ([]*model.Person, error) {
	s.calls++
	return nil, nil
}

func TestSaveRatings_EntryMissingAfterRefetch(t *testing.T) {
	entryID := uuid.New()
	ratingRepo := &stubRatingRepo{}
	entryRepo := &stubEntryRepo{
		entries: []*model.Entry{{ID: entryID}},
		errs:    []error{nil, nil},
	}
	personRepo := &stubPersonRepo{}

	handler := &RatingHandler{
		ratingRepo: ratingRepo,
		entryRepo:  entryRepo,
		personRepo: personRepo,
	}

	form := url.Values{}
	req := httptest.NewRequest(http.MethodPost, "/entries/"+entryID.String()+"/ratings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", entryID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	recorder := httptest.NewRecorder()

	handler.SaveRatings(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
	if entryRepo.calls != 2 {
		t.Fatalf("expected entry repo to be called twice, got %d", entryRepo.calls)
	}
	if personRepo.calls != 0 {
		t.Fatalf("expected persons fetch to be skipped, got %d", personRepo.calls)
	}
	if ratingRepo.upsertCalls != 0 || ratingRepo.deleteCalls != 0 {
		t.Fatalf("expected no rating repo mutations, got upserts=%d deletes=%d", ratingRepo.upsertCalls, ratingRepo.deleteCalls)
	}
}

func TestSaveRatings_InvalidEntryID(t *testing.T) {
	ratingRepo := &stubRatingRepo{}
	entryRepo := &stubEntryRepo{}
	personRepo := &stubPersonRepo{}

	handler := &RatingHandler{
		ratingRepo: ratingRepo,
		entryRepo:  entryRepo,
		personRepo: personRepo,
	}

	req := httptest.NewRequest(http.MethodPost, "/entries/not-a-uuid/ratings", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", "not-a-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	recorder := httptest.NewRecorder()

	handler.SaveRatings(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	if entryRepo.calls != 0 {
		t.Fatalf("expected entry repo not to be called, got %d", entryRepo.calls)
	}
	if personRepo.calls != 0 {
		t.Fatalf("expected persons fetch not to be called, got %d", personRepo.calls)
	}
	if ratingRepo.upsertCalls != 0 || ratingRepo.deleteCalls != 0 {
		t.Fatalf("expected no rating repo mutations, got upserts=%d deletes=%d", ratingRepo.upsertCalls, ratingRepo.deleteCalls)
	}
}

func TestSaveRatings_InvalidForm(t *testing.T) {
	entryID := uuid.New()
	ratingRepo := &stubRatingRepo{}
	entryRepo := &stubEntryRepo{}
	personRepo := &stubPersonRepo{}

	handler := &RatingHandler{
		ratingRepo: ratingRepo,
		entryRepo:  entryRepo,
		personRepo: personRepo,
	}

	req := httptest.NewRequest(http.MethodPost, "/entries/"+entryID.String()+"/ratings", strings.NewReader("rating%ZZ"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", entryID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	recorder := httptest.NewRecorder()

	handler.SaveRatings(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	if entryRepo.calls != 0 {
		t.Fatalf("expected entry repo not to be called, got %d", entryRepo.calls)
	}
	if personRepo.calls != 0 {
		t.Fatalf("expected persons fetch not to be called, got %d", personRepo.calls)
	}
	if ratingRepo.upsertCalls != 0 || ratingRepo.deleteCalls != 0 {
		t.Fatalf("expected no rating repo mutations, got upserts=%d deletes=%d", ratingRepo.upsertCalls, ratingRepo.deleteCalls)
	}
}

func TestSaveRatings_UpsertDeleteAndInvalidScores(t *testing.T) {
	entryID := uuid.New()
	existingPersonID := uuid.New()
	validPersonID := uuid.New()
	invalidPersonID := "not-a-uuid"

	ratingRepo := &stubRatingRepo{}
	entryRepo := &stubEntryRepo{
		entries: []*model.Entry{
			{
				ID: entryID,
				Ratings: []*model.Rating{
					{PersonID: existingPersonID},
				},
			},
			{
				ID: entryID,
			},
		},
		errs: []error{nil, nil},
	}
	personRepo := &stubPersonRepo{}

	handler := &RatingHandler{
		ratingRepo: ratingRepo,
		entryRepo:  entryRepo,
		personRepo: personRepo,
	}

	form := url.Values{}
	form.Set("rating["+existingPersonID.String()+"]", "")
	form.Set("rating["+validPersonID.String()+"]", "8.5")
	form.Set("rating["+uuid.NewString()+"]", "11")
	form.Set("rating["+uuid.NewString()+"]", "-1")
	form.Set("rating["+uuid.NewString()+"]", "abc")
	form.Set("rating["+invalidPersonID+"]", "7")

	req := httptest.NewRequest(http.MethodPost, "/entries/"+entryID.String()+"/ratings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", entryID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	recorder := httptest.NewRecorder()

	handler.SaveRatings(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if entryRepo.calls != 2 {
		t.Fatalf("expected entry repo to be called twice, got %d", entryRepo.calls)
	}
	if personRepo.calls != 1 {
		t.Fatalf("expected persons fetch to be called once, got %d", personRepo.calls)
	}
	if ratingRepo.deleteCalls != 1 {
		t.Fatalf("expected one delete call, got %d", ratingRepo.deleteCalls)
	}
	if ratingRepo.upsertCalls != 1 {
		t.Fatalf("expected one upsert call, got %d", ratingRepo.upsertCalls)
	}
}
