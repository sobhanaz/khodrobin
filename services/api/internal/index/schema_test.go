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
// This pins the fields the UI depends on.
func TestFieldsTheUIDependsOnSurviveTheRoundTrip(t *testing.T) {
	raw := []byte(`{
	  "key":"peugeot/207/base/mt/1404/0",
	  "image":"https://cdn.example/car.webp",
	  "brand":"peugeot","brand_fa":"پژو","model":"207","model_fa":"۲۰۷",
	  "trim":null,"gearbox":"mt","gearbox_fa":"دنده‌ای","year":1404,"km_bucket":0,
	  "offer_count":2,"source_count":2,
	  "median_price":2114000000,"min_price":1660000000,"max_price":2470000000,
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
		"flag_count", "offers",
	} {
		if _, ok := back[field]; !ok {
			t.Errorf("field %q missing from the response", field)
		}
	}
}
