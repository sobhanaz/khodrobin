package index

import (
	"encoding/json"
	"testing"
)

// The Python pipeline owns the index schema and Go re-serializes it to the
// browser. An unknown field is silently dropped by encoding/json, so a field
// added upstream simply vanishes with no error anywhere — which is exactly how
// car photos reached the index and never reached the page.
//
// This has now bitten three times: car photos, then duplicates_collapsed, then
// listings_captured. Each was written by the Python pipeline, logged as present,
// and silently discarded by Go on the way out. So the pin covers Stats too.
func TestFieldsTheUIDependsOnSurviveTheRoundTrip(t *testing.T) {
	raw := []byte(`{
	  "key":"peugeot/207/base/mt/1404/0",
	  "image":"https://cdn.example/car.webp",
	  "brand":"peugeot","brand_fa":"پژو","model":"207","model_fa":"۲۰۷",
	  "trim":null,"gearbox":"mt","gearbox_fa":"دنده‌ای","year":1404,"km_bucket":0,
	  "offer_count":2,"source_count":2,
	  "median_price":2114000000,"median_reliable":true,"min_price":1660000000,"max_price":2470000000,
	  "flag_count":0,
	  "offers":[{
	    "source":"divar","source_fa":"دیوار","title":"۲۰۷",
	    "price":1660000000,"mileage_km":1000,"colour":"سفید","body_status":"بدون رنگ",
	    "city":"تهران","url":"https://divar.ir/v/x",
	    "image":"https://cdn.example/offer.webp",
	    "seen_at":"2026-09-06T00:00:00Z","flags":[],"vs_median_pct":-21.5,
	    "raw_brand":"Peugeot","raw_model":"Peugeot 207","raw_trim":null
	  }]
	}`)

	var spec Spec
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.Image == nil || *spec.Image == "" {
		t.Error("spec image was dropped")
	}
	if !spec.MedianReliable {
		t.Error("median_reliable was dropped")
	}
	if len(spec.Offers) != 1 {
		t.Fatalf("offers = %d, want 1", len(spec.Offers))
	}
	if spec.Offers[0].Image == nil || *spec.Offers[0].Image == "" {
		t.Error("offer image was dropped")
	}

	// Re-encode: what the browser actually receives.
	out, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{
		"key", "image", "brand_fa", "model_fa", "year", "gearbox_fa",
		"offer_count", "source_count", "median_price", "min_price", "max_price",
		"flag_count", "median_reliable", "offers",
	} {
		if _, ok := back[field]; !ok {
			t.Errorf("field %q missing from the response", field)
		}
	}
}

// TestStatsFieldsSurviveTheRoundTrip guards the same failure as above, for the
// numbers an operator reads rather than the ones the page renders.
//
// duplicates_collapsed was dropped exactly this way: the crawler collapsed 3,889
// repeat listings, logged it, and the API reported nothing — so the fix looked
// like it had not shipped.
func TestStatsFieldsSurviveTheRoundTrip(t *testing.T) {
	raw := []byte(`{
	  "listings_captured": 9014,
	  "duplicates_collapsed": 3889,
	  "listings": 5125,
	  "indexed": 2958,
	  "unresolved": 2167,
	  "resolved_pct": 57.7,
	  "specs": 2000,
	  "multi_source_specs": 239,
	  "flagged_offers": 23,
	  "sources": ["bama","divar","hamrah","khodro45"]
	}`)

	var stats Stats
	if err := json.Unmarshal(raw, &stats); err != nil {
		t.Fatal(err)
	}
	if stats.ListingsCaptured != 9014 {
		t.Errorf("listings_captured = %d, want 9014", stats.ListingsCaptured)
	}
	if stats.DuplicatesCollapsed != 3889 {
		t.Errorf("duplicates_collapsed = %d, want 3889", stats.DuplicatesCollapsed)
	}

	out, err := json.Marshal(stats)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{
		"listings_captured", "duplicates_collapsed", "listings", "indexed",
		"resolved_pct", "specs", "multi_source_specs", "flagged_offers", "sources",
	} {
		if _, ok := back[field]; !ok {
			t.Errorf("stats field %q was dropped on the way out", field)
		}
	}
}
