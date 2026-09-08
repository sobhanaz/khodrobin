package lookup

import (
	"testing"

	"github.com/sobhanaz/khodrobin/api/internal/index"
)

// One real URL shape per source, taken from what the crawler actually stores.
func TestParseEverySourceShape(t *testing.T) {
	cases := []struct{ raw, source, id string }{
		{"https://divar.ir/v/سمند-سورن-پلاس-صفر-خشک-۱۴۰۵/gadSNVYG", "divar", "gadSNVYG"},
		{"https://www.sheypoor.com/v/پراید-مدل-1388-هاچبک-کره-انژکتور-467113341.html", "sheypoor", "467113341"},
		{"https://www.sheypoor.com/v/موهاوی-2010-فول-465632926.html", "sheypoor", "465632926"},
		{"https://bama.ir/car/detail-ayxrbse1-dongfeng-h30cross-1398", "bama", "ayxrbse1"},
		{"https://www.hamrah-mechanic.com/cars-for-sale/mg/mg4/3390849/", "hamrah", "3390849"},
		{"https://khodro45.com/used-car/BD4JEC/", "khodro45", "BD4JEC"},
		// No scheme, trailing junk, uppercase host, www — the paste is hostile.
		{"divar.ir/v/samand/gadSNVYG", "divar", "gadSNVYG"},
		{"WWW.BAMA.IR/car/detail-x9-toyota", "bama", "x9"},
		{"  https://khodro45.com/used-car/BD4JEC/  ", "khodro45", "BD4JEC"},
	}
	for _, c := range cases {
		src, id, err := Parse(c.raw)
		if err != nil {
			t.Errorf("Parse(%q) error = %v, want %s/%s", c.raw, err, c.source, c.id)
			continue
		}
		if src != c.source || id != c.id {
			t.Errorf("Parse(%q) = %s/%s, want %s/%s", c.raw, src, id, c.source, c.id)
		}
	}
}

func TestParseRejectsWhatItCannotAttribute(t *testing.T) {
	cases := []string{
		"",
		"https://google.com/search?q=x",
		"https://divar.ir/s/tehran/car",         // a list page, not a listing
		"https://bama.ir/car",                   // no detail token
		"https://bama.ir/magazine/some-article", // right host, wrong section
		"https://www.sheypoor.com/v/بدون-کد.html", // no digit tail
		"https://khodro45.com/car/BD4JEC/",        // wrong section for khodro45
		"https://divar.ir/v/only-slug",            // one segment is never a listing
		"://broken",
	}
	for _, raw := range cases {
		if _, _, err := Parse(raw); err == nil {
			t.Errorf("Parse(%q) = nil error, want ErrUnknown", raw)
		} else if err != ErrUnknown {
			t.Errorf("Parse(%q) error = %v, want ErrUnknown itself", raw, err)
		}
	}
}

func TestSheypoorIdSurvivesAnEditedTitle(t *testing.T) {
	// The slug is the seller's editable title; only the digit tail is the id.
	// Both forms must land on the same id or one ad becomes two offers.
	_, a, _ := Parse("https://www.sheypoor.com/v/موهاوی-2010-465632926.html")
	_, b, _ := Parse("https://www.sheypoor.com/v/موهاوی-2010-فول-و-تمیز-465632926.html")
	if a != b {
		t.Errorf("edited slug changed the id: %q vs %q", a, b)
	}
}

// A tiny index with one spec, three offers from three sources, and one
// flagged offer priced below everything.
func fixture() *index.Index {
	km := 2
	flag := []index.Flag{{Code: "price_outlier", Message: "…"}}
	return &index.Index{Specs: []index.Spec{{
		Key: "peugeot/206/base/mt/1398/2", Brand: "peugeot", BrandFa: "پژو",
		Model: "206", ModelFa: "۲۰۶", Gearbox: "mt", Year: 1398, KmBucket: &km,
		OfferCount: 4, SourceCount: 3, MedianPrice: 1_000 * 1_000_000,
		MinPrice: 90 * 1_000_000, MaxPrice: 1_050 * 1_000_000,
		Offers: []index.Offer{
			{Source: "divar", SourceFa: "دیوار", Price: 1_000 * 1_000_000, URL: "https://divar.ir/v/206/gadSNVYG"},
			{Source: "bama", SourceFa: "باما", Price: 950 * 1_000_000, URL: "https://bama.ir/car/detail-ayxrbse1-206-1398"},
			{Source: "sheypoor", SourceFa: "شیپور", Price: 1_050 * 1_000_000, URL: "https://www.sheypoor.com/v/206-467113341.html"},
			{Source: "khodro45", SourceFa: "خودرو۴۵", Price: 90 * 1_000_000, URL: "https://khodro45.com/used-car/BD4JEC/", Flags: flag},
		},
	}}}
}

func TestFindReturnsTheWholeCluster(t *testing.T) {
	res := Find(fixture(), "divar", "gadSNVYG")
	if res == nil {
		t.Fatal("Find = nil, want a result")
	}
	if res.Spec.Key != "peugeot/206/base/mt/1398/2" {
		t.Errorf("spec = %q, want the 206 spec", res.Spec.Key)
	}
	if res.Offer == nil || res.Offer.Source != "divar" {
		t.Fatalf("offer = %+v, want the divar offer", res.Offer)
	}
	if res.OtherSources != 3 {
		t.Errorf("other_sources = %d, want 3", res.OtherSources)
	}
}

func TestVerdictComparesAgainstTheCheapestCleanOffer(t *testing.T) {
	res := Find(fixture(), "sheypoor", "467113341")
	if res == nil {
		t.Fatal("Find = nil")
	}
	// Sheypoor at 1,050M; cheapest clean is Bama at 950M (the 90M khodro45
	// offer is flagged and must never be the number someone acts on).
	if res.Cheapest == nil || res.Cheapest.Source != "bama" {
		t.Fatalf("cheapest = %+v, want the bama offer", res.Cheapest)
	}
	if res.OverpayPct == nil {
		t.Fatal("overpay_pct = nil, want ~10.5")
	}
	if *res.OverpayPct < 10.4 || *res.OverpayPct > 10.6 {
		t.Errorf("overpay_pct = %.2f, want ≈10.5", *res.OverpayPct)
	}
}

func TestTheCheapestListingHasNoOverpayToReport(t *testing.T) {
	res := Find(fixture(), "bama", "ayxrbse1")
	if res == nil {
		t.Fatal("Find = nil")
	}
	if res.OverpayPct != nil {
		t.Errorf("overpay_pct = %.2f, want nil — the pasted offer IS the cheapest clean one",
			*res.OverpayPct)
	}
	if res.Cheapest == nil {
		t.Error("cheapest_offer should still be named, even when it is the pasted one's neighbour")
	}
}

func TestAFlaggedPriceIsNeverTheComparison(t *testing.T) {
	// Paste the khodro45 outlier itself: it has no clean competitor below it,
	// so the cheapest clean offer is Bama — and the verdict must not pretend
	// the outlier is a bargain against the flagged 90M figure.
	res := Find(fixture(), "khodro45", "BD4JEC")
	if res == nil {
		t.Fatal("Find = nil")
	}
	if res.Cheapest == nil || res.Cheapest.Source != "bama" {
		t.Fatalf("cheapest = %+v, want the bama offer, never a flagged one", res.Cheapest)
	}
}

func TestASingleSourceSpecSaysSo(t *testing.T) {
	idx := fixture()
	idx.Specs[0].Offers = idx.Specs[0].Offers[:1] // only the divar offer remains
	res := Find(idx, "divar", "gadSNVYG")
	if res == nil {
		t.Fatal("Find = nil")
	}
	if res.OtherSources != 0 {
		t.Errorf("other_sources = %d, want 0", res.OtherSources)
	}
	if res.Note != "only" {
		t.Errorf("note = %q, want \"only\"", res.Note)
	}
	if res.Cheapest != nil || res.OverpayPct != nil {
		t.Error("a one-offer spec has no verdict to give")
	}
}

func TestAnOfferNotInTheIndexFindsNothing(t *testing.T) {
	if res := Find(fixture(), "divar", "ZZZZZZZ"); res != nil {
		t.Errorf("Find = %+v, want nil for an unknown id", res)
	}
	// And a URL from a source we do not crawl can never match anything.
	if res := Find(fixture(), "divar", "467113341"); res != nil {
		t.Error("an id must not match across sources")
	}
}

func TestIndexURLsParseBackToTheirIds(t *testing.T) {
	// The matcher re-parses stored URLs; if a stored URL ever fails to parse
	// back, that offer becomes invisible to lookup. This is the coupling test.
	for _, raw := range []string{
		"https://divar.ir/v/206/gadSNVYG",
		"https://bama.ir/car/detail-ayxrbse1-206-1398",
		"https://www.sheypoor.com/v/206-467113341.html",
		"https://www.hamrah-mechanic.com/cars-for-sale/mg/mg4/3390849/",
		"https://khodro45.com/used-car/BD4JEC/",
	} {
		if got := offerID(raw); got == "" {
			t.Errorf("offerID(%q) = empty — this stored offer would be unreachable", raw)
		}
	}
}

func TestPercentagesSurviveDecimals(t *testing.T) {
	// 1,910,000,000 vs 1,900,000,000 is 0.526…% — guard rounding and the
	// float formatting the UI will display. Sheypoor (the fixture's 1,050M)
	// is also priced below the pasted 1,910M, but the verdict compares
	// against the CHEAPEST clean offer, so Bama's 1,900M decides.
	idx := fixture()
	idx.Specs[0].Offers[0].Price = 1_910 * 1_000_000 // divar, pasted
	idx.Specs[0].Offers[1].Price = 1_900 * 1_000_000 // bama, cheapest clean
	idx.Specs[0].Offers[2].Price = 2_500 * 1_000_000 // sheypoor, above both
	res := Find(idx, "divar", "gadSNVYG")
	if res == nil || res.OverpayPct == nil {
		t.Fatal("no verdict")
	}
	if got := *res.OverpayPct; got < 0.5 || got > 0.53 {
		t.Errorf("overpay_pct = %.4f, want ≈0.526", got)
	}
}
