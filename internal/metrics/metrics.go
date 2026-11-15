// Package metrics exposes the survey snapshot as Prometheus gauges.
// Values are recomputed from SQLite on each scrape. They are gauges, not
// counters, because a submission id is upserted in place and the dashboard
// needs the current population rather than a process-lifetime event count.
package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/neuland-ingolstadt/app-feedback/internal/feedback"
)

const scrapeTimeout = 4 * time.Second

type statsFunc func(context.Context) (feedback.Stats, error)

// Handler serves GET /metrics from a private registry so scrapes only see
// survey gauges. Comment text is never exported.
func Handler(stats statsFunc) http.Handler {
	reg := prometheus.NewRegistry()
	reg.MustRegister(&collector{stats: stats})
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorHandling: promhttp.HTTPErrorOnError,
		Timeout:       scrapeTimeout,
	})
}

type collector struct {
	stats statsFunc
}

var (
	submissionsDesc = prometheus.NewDesc(
		"app_feedback_submissions",
		"Stored survey submissions. One series per campaign.",
		[]string{"campaign"}, nil,
	)
	ratingSumDesc = prometheus.NewDesc(
		"app_feedback_rating_sum",
		"Sum of star ratings (1-5) for stored submissions.",
		[]string{"campaign"}, nil,
	)
	ratingDesc = prometheus.NewDesc(
		"app_feedback_rating_responses",
		"Stored submissions by star rating.",
		[]string{"campaign", "rating"}, nil,
	)
	platformDesc = prometheus.NewDesc(
		"app_feedback_platform_responses",
		"Stored submissions by client platform. Empty platform is unknown.",
		[]string{"campaign", "platform"}, nil,
	)
	categoryDesc = prometheus.NewDesc(
		"app_feedback_category_responses",
		"Submissions that selected a category. One submission can increment several categories.",
		[]string{"campaign", "category"}, nil,
	)
	featureDesc = prometheus.NewDesc(
		"app_feedback_feature_interest_responses",
		"Feature-interest answers by idea and level (1 not at all, 4 absolutely).",
		[]string{"campaign", "feature", "level"}, nil,
	)
	featureSumDesc = prometheus.NewDesc(
		"app_feedback_feature_interest_score_sum",
		"Sum of feature-interest levels (1-4) across answers for that idea.",
		[]string{"campaign", "feature"}, nil,
	)
	commentsDesc = prometheus.NewDesc(
		"app_feedback_comments_nonempty",
		"Submissions whose comment is non-empty. Comment text is not exported.",
		[]string{"campaign"}, nil,
	)
	missingDesc = prometheus.NewDesc(
		"app_feedback_missing_features_nonempty",
		"Submissions whose missing-features note is non-empty. Note text is not exported.",
		[]string{"campaign"}, nil,
	)
)

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- submissionsDesc
	ch <- ratingSumDesc
	ch <- ratingDesc
	ch <- platformDesc
	ch <- categoryDesc
	ch <- featureDesc
	ch <- featureSumDesc
	ch <- commentsDesc
	ch <- missingDesc
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), scrapeTimeout)
	defer cancel()
	stats, err := c.stats(ctx)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(submissionsDesc, err)
		return
	}
	for campaign, cs := range stats.Campaigns {
		if cs == nil {
			continue
		}
		must(ch, submissionsDesc, cs.Total, campaign)
		must(ch, ratingSumDesc, cs.RatingSum, campaign)
		for rating, count := range cs.RatingCounts {
			must(ch, ratingDesc, count, campaign, strconv.Itoa(rating))
		}
		for platform, count := range cs.Platforms {
			must(ch, platformDesc, count, campaign, platform)
		}
		for category, count := range cs.Categories {
			must(ch, categoryDesc, count, campaign, category)
		}
		for feature, levels := range cs.FeatureLevels {
			for level, count := range levels {
				must(ch, featureDesc, count, campaign, feature, strconv.Itoa(level))
			}
			must(ch, featureSumDesc, cs.FeatureScoreSum[feature], campaign, feature)
		}
		must(ch, commentsDesc, cs.CommentsNonEmpty, campaign)
		must(ch, missingDesc, cs.MissingFeaturesNonEmpty, campaign)
	}
}

func must(ch chan<- prometheus.Metric, desc *prometheus.Desc, value int, labels ...string) {
	ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(value), labels...)
}
