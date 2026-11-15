package feedback

import "time"

type Rating int

type Category string

type FeatureIdea string

type FeatureInterest int

type FeatureInterestAnswers map[FeatureIdea]FeatureInterest

// Submission matches the Neuland Next client payload (+ abuse fields).
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

	// Abuse fields — not survey answers.
	Challenge string `json:"challenge"`
	Counter   uint64 `json:"counter"`
	Website   string `json:"website,omitempty"` // honeypot; must stay empty
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
	TrustScore      float64                `json:"trustScore"`
	ShadowBanned    bool                   `json:"shadowBanned"`
	UpsertCount     int                    `json:"upsertCount"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

type ChallengeResponse struct {
	Challenge  string    `json:"challenge"`
	Nonce      string    `json:"nonce"`
	Difficulty int       `json:"difficulty"`
	ExpiresAt  time.Time `json:"expiresAt"`
	Algorithm  string    `json:"algorithm"`
}
