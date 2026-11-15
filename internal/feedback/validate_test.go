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
