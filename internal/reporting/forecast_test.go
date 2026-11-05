package reporting

import (
	"reflect"
	"testing"
)

func TestEstimateDemandThreeMonths(t *testing.T) {
	t.Parallel()
	got := EstimateDemand([]MonthUsage{
		{Month: "2026-10", Doses: 30},
		{Month: "2026-09", Doses: 20},
		{Month: "2026-08", Doses: 10},
		{Month: "2026-07", Doses: 99},
	})
	if got.Method != MethodThreeMonthAverage {
		t.Fatalf("method = %s", got.Method)
	}
	if got.Doses != 20 {
		t.Fatalf("doses = %d, want 20", got.Doses)
	}
	want := []string{"2026-10", "2026-09", "2026-08"}
	if !reflect.DeepEqual(got.SampleMonths, want) {
		t.Fatalf("sample months = %v, want %v", got.SampleMonths, want)
	}
}

func TestEstimateDemandRounding(t *testing.T) {
	t.Parallel()
	got := EstimateDemand([]MonthUsage{
		{Month: "2026-10", Doses: 2},
		{Month: "2026-09", Doses: 3},
		{Month: "2026-08", Doses: 3},
	})
	if got.Doses != 3 {
		t.Fatalf("rounded average = %d, want 3", got.Doses)
	}
}

func TestEstimateDemandFallback(t *testing.T) {
	t.Parallel()
	got := EstimateDemand([]MonthUsage{
		{Month: "2026-10", Doses: 17},
		{Month: "2026-09", Doses: 30},
	})
	if got.Method != MethodMostRecentMonth || got.Doses != 17 {
		t.Fatalf("estimate = %+v", got)
	}
	if !reflect.DeepEqual(got.SampleMonths, []string{"2026-10"}) {
		t.Fatalf("sample months = %v", got.SampleMonths)
	}
	empty := EstimateDemand(nil)
	if empty.Method != MethodNoHistory || empty.Doses != 0 {
		t.Fatalf("empty estimate = %+v", empty)
	}
}

func TestSuggestedRequest(t *testing.T) {
	t.Parallel()
	if got := SuggestedRequest(100, 40); got != 60 {
		t.Fatalf("SuggestedRequest(100,40) = %d, want 60", got)
	}
	if got := SuggestedRequest(20, 40); got != 0 {
		t.Fatalf("SuggestedRequest(20,40) = %d, want 0", got)
	}
}
