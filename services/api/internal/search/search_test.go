package search

import (
	"testing"

	"github.com/sobhanaz/khodrobin/api/internal/index"
)

func testVocab() index.Vocab {
	return index.Vocab{
		Brands: []index.Alias{
			{Slug: "peugeot", Fa: "پژو", Aliases: []string{"peugeot", "پژو"}},
			{Slug: "pride", Fa: "پراید", Aliases: []string{"pride", "پراید"}},
			{Slug: "byd", Fa: "بی‌وای‌دی", Aliases: []string{"byd", "بی وای دی"}},
		},
		Models: []index.Alias{
			{Brand: "peugeot", Slug: "206", Fa: "۲۰۶", Aliases: []string{"206", "۲۰۶"}},
			{Brand: "peugeot", Slug: "206-sd", Fa: "۲۰۶ SD", Aliases: []string{"206 sd", "۲۰۶ اس دی"}},
			{Brand: "peugeot", Slug: "405", Fa: "۴۰۵", Aliases: []string{"405"}},
		},
	}
}

func TestParsesARealPersianQuery(t *testing.T) {
	in := ParseQuery("ارزون‌ترین پژو ۲۰۶ بالای مدل ۹۵ زیر ۵۰۰ میلیون", testVocab())
	if in.Brand != "peugeot" || in.Model != "206" {
		t.Fatalf("brand/model = %q/%q, want peugeot/206", in.Brand, in.Model)
	}
	if in.Priority != PriorityCheapest {
		t.Errorf("priority = %q, want %q", in.Priority, PriorityCheapest)
	}
	if in.YearMin != 1395 {
		t.Errorf("year_min = %d, want 1395", in.YearMin)
	}
	if in.PriceMax != 500*million {
		t.Errorf("price_max = %d, want %d", in.PriceMax, 500*million)
	}
}

func TestTwoDigitYearsBecomeJalali(t *testing.T) {
	// Iranians write «مدل ۹۶», not «مدل ۱۳۹۶».
	if got := normalizeYear(96); got != 1396 {
		t.Errorf("96 -> %d, want 1396", got)
	}
	if got := normalizeYear(1402); got != 1402 {
		t.Errorf("1402 -> %d, want 1402", got)
	}
	if got := normalizeYear(2020); got != 1399 {
		t.Errorf("2020 -> %d, want 1399", got)
	}
}

func TestPriceShorthandScales(t *testing.T) {
	// «زیر ۵۰۰» means 500 million tomans, not 500 tomans.
	in := ParseQuery("زیر ۵۰۰", testVocab())
	if in.PriceMax != 500*million {
		t.Errorf("price_max = %d, want %d", in.PriceMax, 500*million)
	}
	in = ParseQuery("تا ۲ میلیارد", testVocab())
	if in.PriceMax != 2*billion {
		t.Errorf("price_max = %d, want %d", in.PriceMax, 2*billion)
	}
}

func TestYearIsNotMistakenForAPrice(t *testing.T) {
	// «بالای ۹۵» is a model year. Reading it as 95 million tomans would filter
	// out every real car.
	in := ParseQuery("پژو ۲۰۶ بالای ۹۵", testVocab())
	if in.YearMin != 1395 {
		t.Errorf("year_min = %d, want 1395", in.YearMin)
	}
	if in.PriceMax != 0 {
		t.Errorf("price_max = %d, want 0", in.PriceMax)
	}
}

func TestModelNumberNeedsADigitBoundary(t *testing.T) {
	// "206" must not match inside a price.
	in := ParseQuery("پژو پارس 2065000000", testVocab())
	if in.Model == "206" {
		t.Error("matched 206 inside a longer number")
	}
}

func TestLongestAliasWins(t *testing.T) {
	if in := ParseQuery("پژو 206 sd", testVocab()); in.Model != "206-sd" {
		t.Errorf("model = %q, want 206-sd", in.Model)
	}
}

func TestUnknownQueryStillReturnsAUsableIntent(t *testing.T) {
	// A parser that gives up is worse than one that shows everything.
	in := ParseQuery("یه ماشین خوب می‌خوام", testVocab())
	if in.Priority == "" {
		t.Error("priority must always be set")
	}
	if in.Source != "rules" {
		t.Errorf("source = %q, want rules", in.Source)
	}
}

func spec(key, brand, model string, year int, median int64, sources, offers, flags int) index.Spec {
	bucket := 0
	return index.Spec{
		Key: key, Brand: brand, Model: model, Year: year, Gearbox: "mt",
		MedianPrice: median, MinPrice: median, MaxPrice: median,
		SourceCount: sources, OfferCount: offers, FlagCount: flags, KmBucket: &bucket,
	}
}

func TestCheapestModePutsThePriceFirst(t *testing.T) {
	idx := &index.Index{Specs: []index.Spec{
		spec("a", "peugeot", "206", 1396, 1_500*million, 1, 1, 0),
		spec("b", "peugeot", "206", 1396, 900*million, 1, 1, 0),
	}}
	got := Run(idx, Intent{Priority: PriorityCheapest}, PriorityCheapest, 10)
	if got.Specs[0].Key != "b" {
		t.Errorf("first = %q, want b (cheaper)", got.Specs[0].Key)
	}
}

func TestCorroborationBreaksATie(t *testing.T) {
	// Same price, same year: the spec confirmed by three marketplaces is the
	// more trustworthy answer, and that is the reason this product exists.
	idx := &index.Index{Specs: []index.Spec{
		spec("one-source", "peugeot", "206", 1396, 1_000*million, 1, 1, 0),
		spec("three-source", "peugeot", "206", 1396, 1_000*million, 3, 6, 0),
	}}
	got := Run(idx, Intent{}, PriorityRelevant, 10)
	if got.Specs[0].Key != "three-source" {
		t.Errorf("first = %q, want three-source", got.Specs[0].Key)
	}
}

func TestContradictoryListingsSink(t *testing.T) {
	idx := &index.Index{Specs: []index.Spec{
		spec("clean", "peugeot", "206", 1396, 1_000*million, 2, 4, 0),
		spec("flagged", "peugeot", "206", 1396, 1_000*million, 2, 4, 4),
	}}
	got := Run(idx, Intent{}, PriorityRelevant, 10)
	if got.Specs[0].Key != "clean" {
		t.Errorf("first = %q, want clean", got.Specs[0].Key)
	}
}

func TestFiltersAreHard(t *testing.T) {
	idx := &index.Index{Specs: []index.Spec{
		spec("old", "peugeot", "206", 1390, 500*million, 1, 1, 0),
		spec("new", "peugeot", "206", 1400, 900*million, 1, 1, 0),
	}}
	got := Run(idx, Intent{YearMin: 1395}, PriorityRelevant, 10)
	if got.Total != 1 || got.Specs[0].Key != "new" {
		t.Errorf("total=%d first=%v, want 1/new", got.Total, got.Specs)
	}
}

func TestBreakdownSumsToScore(t *testing.T) {
	// The UI draws the breakdown as the explanation of the score. If they
	// disagree, the chart is a lie.
	s := spec("x", "peugeot", "206", 1396, 1_000*million, 2, 3, 0)
	total, b := Score(s, Modes[PriorityRelevant], 500*million, 2_000*million)
	sum := 0.0
	for _, v := range b {
		sum += v
	}
	if diff := total - sum; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("score %v != sum of breakdown %v", total, sum)
	}
}
