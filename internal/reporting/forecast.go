// Package reporting contains transparent, pure planning calculations.
package reporting

// MonthUsage is one complete closed month's non-voided dose total.
type MonthUsage struct {
	Month string
	Doses int64
}

// Estimate is a planning estimate with its method and sample window exposed.
type Estimate struct {
	Doses        int64
	Method       string
	SampleMonths []string
}

const (
	MethodThreeMonthAverage = "three_month_average"
	MethodMostRecentMonth   = "most_recent_complete_month"
	MethodNoHistory         = "no_history"
)

// EstimateDemand uses the average of the three most recent complete months
// when available; otherwise it falls back to the most recent complete month.
// History must be ordered newest first. This is an operational estimate, not a
// medical recommendation.
func EstimateDemand(history []MonthUsage) Estimate {
	if len(history) == 0 {
		return Estimate{Method: MethodNoHistory}
	}
	if len(history) >= 3 {
		sample := history[:3]
		var sum int64
		months := make([]string, 0, 3)
		for _, item := range sample {
			sum += item.Doses
			months = append(months, item.Month)
		}
		// Whole-dose average, rounded half up.
		average := (sum + int64(len(sample))/2) / int64(len(sample))
		return Estimate{Doses: average, Method: MethodThreeMonthAverage, SampleMonths: months}
	}
	return Estimate{
		Doses:        history[0].Doses,
		Method:       MethodMostRecentMonth,
		SampleMonths: []string{history[0].Month},
	}
}

// SuggestedRequest is max(0, estimated demand - opening balance).
func SuggestedRequest(estimate, opening int64) int64 {
	if estimate <= opening {
		return 0
	}
	return estimate - opening
}
