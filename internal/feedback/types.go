package feedback

import "time"

type Rating int

type Category string

type FeatureIdea string

type FeatureInterest int

type FeatureInterestAnswers map[FeatureIdea]FeatureInterest

// Submission matches the Neuland Next client payload.
type Submission struct {
	ID              string                 `json:"id"`
	Campaign        string                 `json:"campaign"`
	Rating          Rating                 `json:"rating"`
	Categories      []Category             `json:"categories,omitempty"`
	Comment         *string                `json:"comment,omitempty"`
	FeatureInterest FeatureInterestAnswers `json:"featureInterest,omitempty"`
	MissingFeatures *string                `json:"missingFeatures,omitempty"`
	Platform        string                 `json:"platform,omitempty"`
	AppVersion      string                 `json:"appVersion,omitempty"`
}

type Record struct {
	ID              string                 `json:"id"`
	Campaign        string                 `json:"campaign"`
	Rating          Rating                 `json:"rating"`
	Categories      []Category             `json:"categories"`
	Comment         string                 `json:"comment"`
	FeatureInterest FeatureInterestAnswers `json:"featureInterest"`
	MissingFeatures string                 `json:"missingFeatures"`
	Platform        string                 `json:"platform"`
	AppVersion      string                 `json:"appVersion"`
	ClientIP        string                 `json:"clientIp"`
	UpsertCount     int                    `json:"upsertCount"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}
