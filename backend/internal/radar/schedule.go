package radar

import (
	"time"

	"github.com/robfig/cron/v3"
)

var radarSchedules = map[string]string{
	"regular_collection":    "0 */3 * * *",
	"pre_digest_collection": "40 7,17 * * *",
	"digest":                "0 8,18 * * *",
}

// NextRadarRuns exposes the exact Asia/Shanghai schedule for health checks,
// tests, and the scheduler registration code to share one source of truth.
func NextRadarRuns(after time.Time) (map[string]time.Time, error) {
	runs := make(map[string]time.Time, len(radarSchedules))
	for name, spec := range radarSchedules {
		schedule, err := cron.ParseStandard(spec)
		if err != nil {
			return nil, err
		}
		runs[name] = schedule.Next(after)
	}
	return runs, nil
}

// NewCollectionCron registers only collection jobs. Digest generation remains
// a separate runtime step so collection can ship and be observed first.
func NewCollectionCron(job func()) (*cron.Cron, error) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return nil, err
	}
	scheduler := cron.New(
		cron.WithLocation(location),
		cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)),
	)
	for _, name := range []string{"regular_collection", "pre_digest_collection"} {
		if _, err := scheduler.AddFunc(radarSchedules[name], job); err != nil {
			return nil, err
		}
	}
	return scheduler, nil
}

func NewDigestCron(job func()) (*cron.Cron, error) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return nil, err
	}
	scheduler := cron.New(
		cron.WithLocation(location),
		cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)),
	)
	if _, err := scheduler.AddFunc(radarSchedules["digest"], job); err != nil {
		return nil, err
	}
	return scheduler, nil
}
