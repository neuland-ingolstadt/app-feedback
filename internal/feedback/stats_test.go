package feedback

import "testing"

func TestAggregateCountsSnapshotNotUpserts(t *testing.T) {
	comment := "the timetable jumps"
	missing := "exam registration"
	stats := Aggregate([]Record{
		{
			Campaign:        "campaign-a",
			Rating:          5,
			Categories:      []Category{"timetable", "bugs"},
			Comment:         comment,
			FeatureInterest: FeatureInterestAnswers{"digitalId": 4, "librarySeats": 1},
			MissingFeatures: missing,
			Platform:        "ios",
		},
		{
			Campaign:   "campaign-a",
			Rating:     2,
			Categories: []Category{"food"},
			Platform:   "",
		},
		{
			Campaign:        "campaign-b",
			Rating:          4,
			FeatureInterest: FeatureInterestAnswers{"mensaPayment": 3},
			Platform:        "android",
		},
	}, []string{"campaign-a", "campaign-empty"})

	a := stats.Campaigns["campaign-a"]
	if a == nil || a.Total != 2 || a.RatingSum != 7 {
		t.Fatalf("campaign-a totals: %+v", a)
	}
	if a.RatingCounts[5] != 1 || a.RatingCounts[2] != 1 || a.RatingCounts[1] != 0 {
		t.Fatalf("ratings: %+v", a.RatingCounts)
	}
	if a.Platforms["ios"] != 1 || a.Platforms["unknown"] != 1 || a.Platforms["android"] != 0 {
		t.Fatalf("platforms: %+v", a.Platforms)
	}
	if a.Categories["timetable"] != 1 || a.Categories["bugs"] != 1 || a.Categories["food"] != 1 || a.Categories["map"] != 0 {
		t.Fatalf("categories: %+v", a.Categories)
	}
	if a.FeatureLevels["digitalId"][4] != 1 || a.FeatureScoreSum["digitalId"] != 4 {
		t.Fatalf("digitalId: levels=%v sum=%d", a.FeatureLevels["digitalId"], a.FeatureScoreSum["digitalId"])
	}
	if a.FeatureLevels["librarySeats"][1] != 1 || a.FeatureScoreSum["roomNavigation"] != 0 {
		t.Fatalf("other features: %+v sums=%v", a.FeatureLevels, a.FeatureScoreSum)
	}
	if a.CommentsNonEmpty != 1 || a.MissingFeaturesNonEmpty != 1 {
		t.Fatalf("text counts: comments=%d missing=%d", a.CommentsNonEmpty, a.MissingFeaturesNonEmpty)
	}

	if stats.Campaigns["campaign-b"].Total != 1 || stats.Campaigns["campaign-b"].FeatureScoreSum["mensaPayment"] != 3 {
		t.Fatalf("campaign-b: %+v", stats.Campaigns["campaign-b"])
	}
	empty := stats.Campaigns["campaign-empty"]
	if empty == nil || empty.Total != 0 || empty.RatingCounts[3] != 0 || empty.Categories["bugs"] != 0 {
		t.Fatalf("empty campaign should stay zeroed: %+v", empty)
	}
}

func TestAggregateIgnoresBlankText(t *testing.T) {
	stats := Aggregate([]Record{{
		Campaign:        "campaign-a",
		Rating:          3,
		Comment:         "   ",
		MissingFeatures: "",
		Platform:        "web",
	}}, nil)
	cs := stats.Campaigns["campaign-a"]
	if cs.CommentsNonEmpty != 0 || cs.MissingFeaturesNonEmpty != 0 || cs.Platforms["web"] != 1 {
		t.Fatalf("unexpected: %+v", cs)
	}
}
