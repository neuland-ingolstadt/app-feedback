package feedback

import "strings"

// Stats is the current survey snapshot, keyed by campaign.
type Stats struct {
	Campaigns map[string]*CampaignStats
}

// CampaignStats holds gauge inputs for one campaign. Maps include every known
// dimension at zero so a scrape before the first answer still renders.
type CampaignStats struct {
	Total                   int
	RatingCounts            map[int]int
	RatingSum               int
	Platforms               map[string]int
	Categories              map[string]int
	FeatureLevels           map[string]map[int]int
	FeatureScoreSum         map[string]int
	CommentsNonEmpty        int
	MissingFeaturesNonEmpty int
}

// Aggregate folds stored rows into dashboard gauges. campaigns lists ids that
// must appear even when they have no rows (the deployment allowlist).
func Aggregate(rows []Record, campaigns []string) Stats {
	out := Stats{Campaigns: make(map[string]*CampaignStats, len(campaigns)+1)}
	for _, campaign := range campaigns {
		if campaign == "" {
			continue
		}
		out.campaign(campaign)
	}
	for _, row := range rows {
		if row.Campaign == "" {
			continue
		}
		cs := out.campaign(row.Campaign)
		cs.Total++
		if row.Rating >= MinRating && row.Rating <= MaxRating {
			cs.RatingCounts[int(row.Rating)]++
			cs.RatingSum += int(row.Rating)
		}
		platform := row.Platform
		if platform == "" {
			platform = UnknownPlatform
		}
		cs.Platforms[platform]++
		for _, category := range row.Categories {
			cs.Categories[string(category)]++
		}
		for idea, level := range row.FeatureInterest {
			if level < MinFeatureInterest || level > MaxFeatureInterest {
				continue
			}
			feature := string(idea)
			levels := cs.FeatureLevels[feature]
			if levels == nil {
				levels = newFeatureLevels()
				cs.FeatureLevels[feature] = levels
			}
			levels[int(level)]++
			cs.FeatureScoreSum[feature] += int(level)
		}
		if strings.TrimSpace(row.Comment) != "" {
			cs.CommentsNonEmpty++
		}
		if strings.TrimSpace(row.MissingFeatures) != "" {
			cs.MissingFeaturesNonEmpty++
		}
	}
	return out
}

func (s Stats) campaign(name string) *CampaignStats {
	cs := s.Campaigns[name]
	if cs == nil {
		cs = newCampaignStats()
		s.Campaigns[name] = cs
	}
	return cs
}

func newCampaignStats() *CampaignStats {
	cs := &CampaignStats{
		RatingCounts:    make(map[int]int, MaxRating),
		Platforms:       make(map[string]int, len(Platforms)+1),
		Categories:      make(map[string]int, len(Categories)),
		FeatureLevels:   make(map[string]map[int]int, len(Features)),
		FeatureScoreSum: make(map[string]int, len(Features)),
	}
	for rating := int(MinRating); rating <= int(MaxRating); rating++ {
		cs.RatingCounts[rating] = 0
	}
	for _, platform := range Platforms {
		cs.Platforms[platform] = 0
	}
	for _, category := range Categories {
		cs.Categories[string(category)] = 0
	}
	for _, feature := range Features {
		name := string(feature)
		cs.FeatureLevels[name] = newFeatureLevels()
		cs.FeatureScoreSum[name] = 0
	}
	return cs
}

func newFeatureLevels() map[int]int {
	levels := make(map[int]int, MaxFeatureInterest)
	for level := int(MinFeatureInterest); level <= int(MaxFeatureInterest); level++ {
		levels[level] = 0
	}
	return levels
}
