package feedback

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	idPattern       = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)
	campaignPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{3,64}$`)

	allowedCategories = map[Category]struct{}{
		"bugs": {}, "performance": {}, "login": {}, "usability": {},
		"design": {}, "missingFeature": {}, "timetable": {}, "food": {},
		"grades": {}, "map": {},
	}
	allowedFeatures = map[FeatureIdea]struct{}{
		"digitalId": {}, "mensaPayment": {}, "librarySeats": {}, "roomNavigation": {},
	}
	allowedPlatforms = map[string]struct{}{
		"ios": {}, "android": {}, "web": {},
	}
)

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
	if s.Rating < 1 || s.Rating > 5 {
		return fmt.Errorf("rating must be 1..5")
	}
	if s.Challenge == "" {
		return fmt.Errorf("missing challenge")
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
		if level < 1 || level > 4 {
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
