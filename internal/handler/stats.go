package handler

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"sort"

	"github.com/drywaters/dejaview/internal/model"
	"github.com/drywaters/dejaview/internal/repository"
	"github.com/drywaters/dejaview/internal/ui/pages"
	"github.com/google/uuid"
)

// StatsHandler handles the statistics dashboard
type StatsHandler struct {
	statsRepo *repository.StatsRepository
}

// NewStatsHandler creates a new StatsHandler
func NewStatsHandler(statsRepo *repository.StatsRepository) *StatsHandler {
	return &StatsHandler{
		statsRepo: statsRepo,
	}
}

// StatsPage renders the statistics dashboard
func (h *StatsHandler) StatsPage(w http.ResponseWriter, r *http.Request) {
	statsData, err := h.buildStatsData(r.Context())
	if err != nil {
		slog.Error("failed to build stats data", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	pages.StatsPage(statsData).Render(r.Context(), w)
}

// buildStatsData aggregates all statistics and calculates awards
func (h *StatsHandler) buildStatsData(ctx context.Context) (*model.StatsData, error) {
	// Get all persons for lookup
	persons, err := h.statsRepo.GetAllPersons(ctx)
	if err != nil {
		return nil, fmt.Errorf("get persons: %w", err)
	}

	// Get current group
	currentGroup, err := h.statsRepo.GetCurrentGroup(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current group: %w", err)
	}

	// Get advantage holder
	advantageHolder, advantageGroup, err := h.statsRepo.GetAdvantageHolder(ctx, currentGroup)
	if err != nil {
		return nil, fmt.Errorf("get advantage holder: %w", err)
	}

	// Get all the raw stats
	pickPositionStats, err := h.statsRepo.GetPickPositionStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get pick position stats: %w", err)
	}

	ratingStats, err := h.statsRepo.GetRatingStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get rating stats: %w", err)
	}

	deviationStats, err := h.statsRepo.GetDeviationStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get deviation stats: %w", err)
	}

	selfRatingStats, err := h.statsRepo.GetSelfRatingStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get self rating stats: %w", err)
	}

	pickMetadataStats, err := h.statsRepo.GetPickMetadataStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get pick metadata stats: %w", err)
	}

	movieVariance, err := h.statsRepo.GetMovieRatingVariance(ctx)
	if err != nil {
		return nil, fmt.Errorf("get movie variance: %w", err)
	}

	pickCounts, err := h.statsRepo.GetPickCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("get pick counts: %w", err)
	}

	totalWatched, totalRuntime, totalGroups, fullyRated, err := h.statsRepo.GetSummaryStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get summary stats: %w", err)
	}

	// Build person stats map
	personStatsMap := h.buildPersonStatsMap(
		persons,
		pickPositionStats,
		ratingStats,
		deviationStats,
		selfRatingStats,
		pickMetadataStats,
		pickCounts,
	)

	// Calculate awards
	awards := h.calculateAwards(personStatsMap, persons)

	// Calculate movie awards
	movieAwards := h.calculateMovieAwards(movieVariance)

	// Build leaderboards
	leaderboards := h.buildLeaderboards(personStatsMap, persons)

	// Convert person stats map to slice
	var personStatsList []model.PersonStats
	for _, ps := range personStatsMap {
		personStatsList = append(personStatsList, ps)
	}

	return &model.StatsData{
		AdvantageHolder:       advantageHolder,
		AdvantageGroup:        advantageGroup,
		Awards:                awards,
		MovieAwards:           movieAwards,
		Leaderboards:          leaderboards,
		PersonStats:           personStatsList,
		TotalMoviesWatched:    totalWatched,
		TotalWatchTimeMinutes: totalRuntime,
		TotalGroups:           totalGroups,
		FullyRatedMovies:      fullyRated,
	}, nil
}

// buildPersonStatsMap combines all stats into PersonStats structs
func (h *StatsHandler) buildPersonStatsMap(
	persons map[uuid.UUID]*model.Person,
	pickPositionStats []model.PickPositionStats,
	ratingStats []model.RatingStats,
	deviationStats []model.DeviationStats,
	selfRatingStats []model.SelfRatingStats,
	pickMetadataStats []model.PickMetadataStats,
	pickCounts map[uuid.UUID]int,
) map[uuid.UUID]model.PersonStats {
	statsMap := make(map[uuid.UUID]model.PersonStats)

	// Initialize with persons
	for id, p := range persons {
		statsMap[id] = model.PersonStats{
			Person:     p,
			TotalPicks: pickCounts[id],
		}
	}

	// Add pick position stats
	for _, pps := range pickPositionStats {
		if ps, ok := statsMap[pps.PersonID]; ok {
			ps.FirstPickCount = pps.FirstPickCount
			ps.LastPickCount = pps.LastPickCount
			statsMap[pps.PersonID] = ps
		}
	}

	// Add rating stats
	for _, rs := range ratingStats {
		if ps, ok := statsMap[rs.PersonID]; ok {
			ps.AvgRatingGiven = rs.AvgRatingGiven
			ps.AvgRatingReceived = rs.AvgRatingReceived
			ps.RatingStdDev = rs.RatingStdDev
			ps.MoviesRated = rs.TotalRatingsGiven
			statsMap[rs.PersonID] = ps
		}
	}

	// Add deviation stats
	for _, ds := range deviationStats {
		if ps, ok := statsMap[ds.PersonID]; ok {
			ps.AvgDeviationFromGroup = ds.AvgDeviation
			statsMap[ds.PersonID] = ps
		}
	}

	// Add self rating stats
	for _, srs := range selfRatingStats {
		if ps, ok := statsMap[srs.PersonID]; ok {
			ps.SelfLowestCount = srs.SelfLowestCount
			statsMap[srs.PersonID] = ps
		}
	}

	// Add pick metadata stats
	for _, pms := range pickMetadataStats {
		if ps, ok := statsMap[pms.PersonID]; ok {
			ps.TotalRuntimePicked = pms.TotalRuntime
			ps.AvgReleaseYear = pms.AvgReleaseYear
			statsMap[pms.PersonID] = ps
		}
	}

	return statsMap
}

// calculateAwards determines who wins each award
func (h *StatsHandler) calculateAwards(statsMap map[uuid.UUID]model.PersonStats, persons map[uuid.UUID]*model.Person) []model.Award {
	var awards []model.Award

	// The Headliner - most first picks
	if winner, value := h.findMax(statsMap, func(ps model.PersonStats) float64 {
		return float64(ps.FirstPickCount)
	}); winner != nil && value > 0 {
		awards = append(awards, model.Award{
			ID:          "headliner",
			Title:       "The Headliner",
			Description: "Always opening night material",
			Icon:        "\U0001F451", // crown
			Winner:      winner,
			Value:       fmt.Sprintf("%d first picks", int(value)),
		})
	}

	// The Biggest Loser - most last picks
	if winner, value := h.findMax(statsMap, func(ps model.PersonStats) float64 {
		return float64(ps.LastPickCount)
	}); winner != nil && value > 0 {
		awards = append(awards, model.Award{
			ID:          "biggest_loser",
			Title:       "The Biggest Loser",
			Description: "The comeback kid (3 entries next time!)",
			Icon:        "\U0001F3B0", // slot machine
			Winner:      winner,
			Value:       fmt.Sprintf("%d last picks", int(value)),
		})
	}

	// Corporate Darling - highest avg rating received on picks
	if winner, value := h.findMax(statsMap, func(ps model.PersonStats) float64 {
		if ps.TotalPicks == 0 {
			return 0
		}
		return ps.AvgRatingReceived
	}); winner != nil && value > 0 {
		awards = append(awards, model.Award{
			ID:          "corporate_darling",
			Title:       "Corporate Darling",
			Description: "The family always approves",
			Icon:        "\U0001F4BC", // briefcase
			Winner:      winner,
			Value:       fmt.Sprintf("%.1f avg on picks", value),
		})
	}

	// Harsh Critic - lowest avg rating given
	if winner, value := h.findMin(statsMap, func(ps model.PersonStats) float64 {
		if ps.MoviesRated == 0 {
			return 999
		}
		return ps.AvgRatingGiven
	}); winner != nil && value < 999 {
		awards = append(awards, model.Award{
			ID:          "harsh_critic",
			Title:       "The Harsh Critic",
			Description: "Tough crowd, party of one",
			Icon:        "\U0001F9D0", // monocle
			Winner:      winner,
			Value:       fmt.Sprintf("%.1f avg given", value),
		})
	}

	// Easy Pleaser - highest avg rating given
	if winner, value := h.findMax(statsMap, func(ps model.PersonStats) float64 {
		if ps.MoviesRated == 0 {
			return 0
		}
		return ps.AvgRatingGiven
	}); winner != nil && value > 0 {
		awards = append(awards, model.Award{
			ID:          "easy_pleaser",
			Title:       "The Easy Pleaser",
			Description: "Everything's a 10 with popcorn",
			Icon:        "\U0001F60A", // smiling face
			Winner:      winner,
			Value:       fmt.Sprintf("%.1f avg given", value),
		})
	}

	// Critical Outlier - highest avg deviation from group
	if winner, value := h.findMax(statsMap, func(ps model.PersonStats) float64 {
		return ps.AvgDeviationFromGroup
	}); winner != nil && value > 0 {
		awards = append(awards, model.Award{
			ID:          "critical_outlier",
			Title:       "The Critical Outlier",
			Description: "Marching to their own projector",
			Icon:        "\U0001F3AD", // theater masks
			Winner:      winner,
			Value:       fmt.Sprintf("%.1f points different on average", value),
		})
	}

	// Movie Masochist - most times rating own pick lowest
	if winner, value := h.findMax(statsMap, func(ps model.PersonStats) float64 {
		return float64(ps.SelfLowestCount)
	}); winner != nil && value > 0 {
		awards = append(awards, model.Award{
			ID:          "movie_masochist",
			Title:       "The Movie Masochist",
			Description: "Picks 'em, then roasts 'em",
			Icon:        "\U0001F605", // sweating smile
			Winner:      winner,
			Value:       fmt.Sprintf("%d times", int(value)),
		})
	}

	// The Steady Hand - lowest rating stddev (most consistent)
	if winner, value := h.findMin(statsMap, func(ps model.PersonStats) float64 {
		if ps.MoviesRated == 0 {
			return 999
		}
		return ps.RatingStdDev
	}); winner != nil && value < 999 {
		awards = append(awards, model.Award{
			ID:          "steady_hand",
			Title:       "The Steady Hand",
			Description: "You always know what you're getting",
			Icon:        "\U0001F4CF", // ruler
			Winner:      winner,
			Value:       fmt.Sprintf("%.1f rating spread", value),
		})
	}

	// The Wildcard - highest rating stddev (most inconsistent)
	if winner, value := h.findMax(statsMap, func(ps model.PersonStats) float64 {
		if ps.MoviesRated == 0 {
			return 0
		}
		return ps.RatingStdDev
	}); winner != nil && value > 0 {
		awards = append(awards, model.Award{
			ID:          "wildcard",
			Title:       "The Wildcard",
			Description: "10 or 2, no in-between",
			Icon:        "\U0001F3B2", // dice
			Winner:      winner,
			Value:       fmt.Sprintf("%.1f rating spread", value),
		})
	}

	// Throwback Royalty - oldest avg release year on picks
	if winner, value := h.findMin(statsMap, func(ps model.PersonStats) float64 {
		if ps.TotalPicks == 0 || ps.AvgReleaseYear == 0 {
			return 9999
		}
		return ps.AvgReleaseYear
	}); winner != nil && value < 9999 {
		awards = append(awards, model.Award{
			ID:          "throwback_royalty",
			Title:       "Throwback Royalty",
			Description: "They don't make 'em like they used to",
			Icon:        "\U0001F4FC", // VHS
			Winner:      winner,
			Value:       fmt.Sprintf("avg year: %.0f", value),
		})
	}

	// Fresh Picker - newest avg release year on picks
	if winner, value := h.findMax(statsMap, func(ps model.PersonStats) float64 {
		if ps.TotalPicks == 0 {
			return 0
		}
		return ps.AvgReleaseYear
	}); winner != nil && value > 0 {
		awards = append(awards, model.Award{
			ID:          "fresh_picker",
			Title:       "The Fresh Picker",
			Description: "First in line at the multiplex",
			Icon:        "\U0001F37F", // popcorn
			Winner:      winner,
			Value:       fmt.Sprintf("avg year: %.0f", value),
		})
	}

	// Marathon Runner - longest total runtime on picks
	if winner, value := h.findMax(statsMap, func(ps model.PersonStats) float64 {
		return float64(ps.TotalRuntimePicked)
	}); winner != nil && value > 0 {
		hours := int(value) / 60
		mins := int(value) % 60
		awards = append(awards, model.Award{
			ID:          "marathon_runner",
			Title:       "The Marathon Runner",
			Description: "Bladder of steel",
			Icon:        "\u23F1\uFE0F", // stopwatch
			Winner:      winner,
			Value:       fmt.Sprintf("%dh %dm total", hours, mins),
		})
	}

	return awards
}

// calculateMovieAwards determines which movies win the movie awards
func (h *StatsHandler) calculateMovieAwards(movieVariance []model.MovieWithStats) []model.MovieAward {
	var awards []model.MovieAward

	if len(movieVariance) == 0 {
		return awards
	}

	// The Hype Train - highest variance (most divisive)
	hypeTrain := movieVariance[0] // already sorted by stddev DESC
	if hypeTrain.RatingStdDev > 0 {
		awards = append(awards, model.MovieAward{
			ID:          "hype_train",
			Title:       "The Hype Train",
			Description: "Love it or hate it",
			Icon:        "\U0001F682", // train
			Movie:       hypeTrain.Movie,
			Entry:       hypeTrain.Entry,
			Value:       fmt.Sprintf("Rating spread: %.1f", hypeTrain.RatingStdDev),
		})
	}

	// The Unifier - lowest variance (everyone agreed)
	unifier := movieVariance[len(movieVariance)-1]
	if len(movieVariance) > 1 {
		awards = append(awards, model.MovieAward{
			ID:          "unifier",
			Title:       "The Unifier",
			Description: "Rare family consensus",
			Icon:        "\U0001F91D", // handshake
			Movie:       unifier.Movie,
			Entry:       unifier.Entry,
			Value:       fmt.Sprintf("Rating spread: %.1f", unifier.RatingStdDev),
		})
	}

	return awards
}

// buildLeaderboards creates the leaderboard data
func (h *StatsHandler) buildLeaderboards(statsMap map[uuid.UUID]model.PersonStats, persons map[uuid.UUID]*model.Person) []model.Leaderboard {
	var leaderboards []model.Leaderboard

	// Generosity Index (avg rating given)
	var generosityEntries []model.LeaderboardEntry
	var maxGenerosity float64
	for _, ps := range statsMap {
		if ps.MoviesRated > 0 {
			generosityEntries = append(generosityEntries, model.LeaderboardEntry{
				Person: ps.Person,
				Value:  ps.AvgRatingGiven,
				Label:  fmt.Sprintf("%.1f", ps.AvgRatingGiven),
			})
			if ps.AvgRatingGiven > maxGenerosity {
				maxGenerosity = ps.AvgRatingGiven
			}
		}
	}
	sort.Slice(generosityEntries, func(i, j int) bool {
		return generosityEntries[i].Value > generosityEntries[j].Value
	})
	if len(generosityEntries) > 0 {
		leaderboards = append(leaderboards, model.Leaderboard{
			Title:    "Generosity Index",
			Icon:     "\U0001F381", // gift
			Entries:  generosityEntries,
			MaxValue: maxGenerosity,
		})
	}

	// Pick Success Rate (avg rating received on picks)
	var successEntries []model.LeaderboardEntry
	var maxSuccess float64
	for _, ps := range statsMap {
		if ps.TotalPicks > 0 && ps.AvgRatingReceived > 0 {
			successEntries = append(successEntries, model.LeaderboardEntry{
				Person: ps.Person,
				Value:  ps.AvgRatingReceived,
				Label:  fmt.Sprintf("%.1f", ps.AvgRatingReceived),
			})
			if ps.AvgRatingReceived > maxSuccess {
				maxSuccess = ps.AvgRatingReceived
			}
		}
	}
	sort.Slice(successEntries, func(i, j int) bool {
		return successEntries[i].Value > successEntries[j].Value
	})
	if len(successEntries) > 0 {
		leaderboards = append(leaderboards, model.Leaderboard{
			Title:    "Pick Success Rate",
			Icon:     "\U0001F3AF", // target
			Entries:  successEntries,
			MaxValue: maxSuccess,
		})
	}

	// Movies Picked
	var pickEntries []model.LeaderboardEntry
	var maxPicks float64
	for _, ps := range statsMap {
		if ps.TotalPicks > 0 {
			pickEntries = append(pickEntries, model.LeaderboardEntry{
				Person: ps.Person,
				Value:  float64(ps.TotalPicks),
				Label:  fmt.Sprintf("%d", ps.TotalPicks),
			})
			if float64(ps.TotalPicks) > maxPicks {
				maxPicks = float64(ps.TotalPicks)
			}
		}
	}
	sort.Slice(pickEntries, func(i, j int) bool {
		return pickEntries[i].Value > pickEntries[j].Value
	})
	if len(pickEntries) > 0 {
		leaderboards = append(leaderboards, model.Leaderboard{
			Title:    "Total Picks",
			Icon:     "\U0001F3AC", // clapperboard
			Entries:  pickEntries,
			MaxValue: maxPicks,
		})
	}

	return leaderboards
}

// findMax finds the person with the maximum value for the given metric
func (h *StatsHandler) findMax(statsMap map[uuid.UUID]model.PersonStats, metric func(model.PersonStats) float64) (*model.Person, float64) {
	var winner *model.Person
	var maxVal float64 = -1

	for _, ps := range statsMap {
		val := metric(ps)
		if val > maxVal {
			maxVal = val
			winner = ps.Person
		}
	}

	return winner, maxVal
}

// findMin finds the person with the minimum value for the given metric
func (h *StatsHandler) findMin(statsMap map[uuid.UUID]model.PersonStats, metric func(model.PersonStats) float64) (*model.Person, float64) {
	var winner *model.Person
	minVal := math.Inf(1)

	for _, ps := range statsMap {
		val := metric(ps)
		if val < minVal {
			minVal = val
			winner = ps.Person
		}
	}

	return winner, minVal
}
