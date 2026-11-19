package feedback

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	MinRating          Rating          = 1
	MaxRating          Rating          = 5
	MinFeatureInterest FeatureInterest = 1
	MaxFeatureInterest FeatureInterest = 4
)

const UnknownPlatform = "unknown"

// Categories, Features, and Platforms are the survey dimensions the API accepts.
// Metrics export emits every value, including zeroes, so dashboards stay stable.
var (
	Categories = []Category{
		"bugs", "performance", "login", "usability",
		"design", "missingFeature", "timetable", "food",
		"grades", "map",
	}
	Features = []FeatureIdea{
		"digitalId", "mensaPayment", "librarySeats", "roomNavigation",
		"studentExchange", "campusUpdates", "eventSignup", "mensaRatings",
	}
	Platforms = []string{"ios", "android", "web"}
)

var (
	idPattern       = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)
	campaignPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{3,64}$`)

	allowedCategories map[Category]struct{}
	allowedFeatures   map[FeatureIdea]struct{}
	allowedPlatforms  map[string]struct{}
)

func init() {
	allowedCategories = make(map[Category]struct{}, len(Categories))
	for _, c := range Categories {
		allowedCategories[c] = struct{}{}
	}
	allowedFeatures = make(map[FeatureIdea]struct{}, len(Features))
	for _, f := range Features {
		allowedFeatures[f] = struct{}{}
	}
	allowedPlatforms = make(map[string]struct{}, len(Platforms))
	for _, p := range Platforms {
		allowedPlatforms[p] = struct{}{}
	}
}

func ValidateSubmission(s Submission, maxCommentBytes int, allowedCampaigns []string) error {
	if !idPattern.MatchString(s.ID) {
		return fmt.Errorf("invalid id")
	}
	if !campaignPattern.MatchString(s.Campaign) {
		return fmt.Errorf("invalid campaign")
	}
	if len(allowedCampaigns) > 0 && !contains(allowedCampaigns, s.Campaign) {
		return fmt.Errorf("campaign not allowed")
	}
	if s.Rating < MinRating || s.Rating > MaxRating {
		return fmt.Errorf("rating must be 1..5")
	}
	seen := make(map[Category]struct{}, len(s.Categories))
	for _, c := range s.Categories {
		if _, ok := allowedCategories[c]; !ok {
			return fmt.Errorf("invalid category %q", c)
		}
		if _, dup := seen[c]; dup {
			return fmt.Errorf("duplicate category %q", c)
		}
		seen[c] = struct{}{}
	}
	for idea, level := range s.FeatureInterest {
		if _, ok := allowedFeatures[idea]; !ok {
			return fmt.Errorf("invalid feature idea %q", idea)
		}
		if level < MinFeatureInterest || level > MaxFeatureInterest {
			return fmt.Errorf("feature interest must be 1..4")
		}
	}
	if s.Platform != "" {
		if _, ok := allowedPlatforms[s.Platform]; !ok {
			return fmt.Errorf("invalid platform")
		}
	}
	if err := checkText(s.Comment, maxCommentBytes, "comment"); err != nil {
		return err
	}
	if err := checkText(s.MissingFeatures, maxCommentBytes, "missingFeatures"); err != nil {
		return err
	}
	if utf8.RuneCountInString(s.AppVersion) > 32 {
		return fmt.Errorf("appVersion too long")
	}
	return nil
}

func checkText(v *string, maxBytes int, field string) error {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	*v = s
	if s == "" {
		*v = ""
		return nil
	}
	if len(s) > maxBytes {
		return fmt.Errorf("%s too long", field)
	}
	if !utf8.ValidString(s) {
		return fmt.Errorf("%s is not valid utf-8", field)
	}
	return nil
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
