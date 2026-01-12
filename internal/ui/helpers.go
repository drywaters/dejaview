package ui

import (
	"fmt"
	"strconv"

	"github.com/drywaters/dejaview/internal/model"
	"github.com/google/uuid"
)

func IntToStr(n int) string {
	return strconv.Itoa(n)
}

func FormatFloat(f float64) string {
	return fmt.Sprintf("%.1f", f)
}

func GetRatingScore(entry *model.Entry, personID uuid.UUID) *float64 {
	for _, r := range entry.Ratings {
		if r.PersonID == personID {
			return &r.Score
		}
	}
	return nil
}
