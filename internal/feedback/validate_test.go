package feedback

import "testing"

func TestValidateSubmissionOK(t *testing.T) {
	comment := "great app"
	err := ValidateSubmission(Submission{
		ID:         "abc12345",
		Campaign:   "feedback-campaign-2026-10-29_18-55",
		Rating:     5,
		Categories: []Category{"timetable"},
		Comment:    &comment,
		Platform:   "ios",
	}, 4000, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsBadRating(t *testing.T) {
	err := ValidateSubmission(Submission{
		ID:       "abc12345",
		Campaign: "camp",
		Rating:   9,
	}, 4000, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateCampaignAllowlist(t *testing.T) {
	err := ValidateSubmission(Submission{
		ID:       "abc12345",
		Campaign: "other",
		Rating:   3,
	}, 4000, []string{"allowed-one"})
	if err == nil {
		t.Fatal("expected allowlist error")
	}
}

func TestValidateAcceptsExtendedFeatureIdeas(t *testing.T) {
	err := ValidateSubmission(Submission{
		ID:       "abc12345",
		Campaign: "feedback-campaign-2026-10-29_18-55",
		Rating:   4,
		FeatureInterest: FeatureInterestAnswers{
			"digitalId":       1,
			"mensaPayment":    2,
			"librarySeats":    3,
			"roomNavigation":  4,
			"studentExchange": 1,
			"campusUpdates":   2,
			"eventSignup":     3,
			"mensaRatings":    4,
		},
		Platform: "android",
	}, 4000, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsUnknownFeatureIdea(t *testing.T) {
	err := ValidateSubmission(Submission{
		ID:       "abc12345",
		Campaign: "camp",
		Rating:   3,
		FeatureInterest: FeatureInterestAnswers{
			"notARealFeature": 2,
		},
	}, 4000, nil)
	if err == nil {
		t.Fatal("expected invalid feature idea error")
	}
}

func TestValidateAcceptsHighlightCategories(t *testing.T) {
	err := ValidateSubmission(Submission{
		ID:       "abc12345",
		Campaign: "feedback-campaign-2026-10-29_18-55",
		Rating:   5,
		Categories: []Category{
			"timetable", "food", "grades", "map",
			"events", "news", "sports", "calendar",
			"design", "performance",
		},
		Platform: "ios",
	}, 4000, nil)
	if err != nil {
		t.Fatal(err)
	}
}
