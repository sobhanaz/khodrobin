package search

import (
	"net/url"
	"reflect"
	"sort"
	"testing"

	"github.com/sobhanaz/khodrobin/api/internal/index"
)

// parseRaw builds url.Values from a literal query string, so the table rows
// read exactly like the URLs the web client actually sends — including the
// present-but-empty form ("price_max="), which url.Values literals make too
// easy to get wrong.
func parseRaw(t *testing.T, raw string) url.Values {
	t.Helper()
	v, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("bad raw query %q: %v", raw, err)
	}
	return v
}

func TestOverrideMergesFieldByField(t *testing.T) {
	cases := []struct {
		name  string
		q     string
		raw   string
		check func(t *testing.T, in Intent)
	}{
		{
			name: "present wins over text",
			q:    "پژو زیر ۵۰۰ میلیون",
			raw:  "price_max=700000000",
			check: func(t *testing.T, in Intent) {
				if in.PriceMax != 700_000_000 {
					t.Errorf("price_max = %d, want 700000000", in.PriceMax)
				}
			},
		},
		{
			name: "present but empty clears what the text said",
			q:    "پژو زیر ۵۰۰ میلیون",
			raw:  "price_max=",
			check: func(t *testing.T, in Intent) {
				if in.PriceMax != 0 {
					t.Errorf("price_max = %d, want 0 (cleared)", in.PriceMax)
				}
			},
		},
		{
			name: "garbage number is treated as absent, not as a clear",
			q:    "پژو زیر ۵۰۰ میلیون",
			raw:  "price_max=abc",
			check: func(t *testing.T, in Intent) {
				if in.PriceMax != 500*million {
					t.Errorf("price_max = %d, want the parsed %d", in.PriceMax, 500*million)
				}
			},
		},
		{
			name: "negative number is treated as absent",
			q:    "پژو زیر ۵۰۰ میلیون",
			raw:  "price_max=-5",
			check: func(t *testing.T, in Intent) {
				if in.PriceMax != 500*million {
					t.Errorf("price_max = %d, want the parsed %d", in.PriceMax, 500*million)
				}
			},
		},
		{
			name: "brand override wins and its label follows the vocab",
			q:    "پژو ۲۰۶",
			raw:  "brand=kia",
			check: func(t *testing.T, in Intent) {
				if in.Brand != "kia" || in.BrandFa != "کیا" {
					t.Errorf("brand = %q/%q, want kia/کیا", in.Brand, in.BrandFa)
				}
				if in.Model != "206" {
					t.Errorf("model = %q, want the parsed 206 untouched", in.Model)
				}
			},
		},
		{
			name: "cleared brand clears its label too",
			q:    "پژو ۲۰۶",
			raw:  "brand=",
			check: func(t *testing.T, in Intent) {
				if in.Brand != "" || in.BrandFa != "" {
					t.Errorf("brand = %q/%q, want both empty", in.Brand, in.BrandFa)
				}
			},
		},
		{
			name: "model override resolves its label within the brand",
			q:    "پژو ۲۰۶",
			raw:  "model=405",
			check: func(t *testing.T, in Intent) {
				if in.Model != "405" || in.ModelFa != "۴۰۵" {
					t.Errorf("model = %q/%q, want 405/۴۰۵", in.Model, in.ModelFa)
				}
			},
		},
		{
			name: "gearbox override beats the text",
			q:    "پژو اتومات",
			raw:  "gearbox=mt",
			check: func(t *testing.T, in Intent) {
				if in.Gearbox != "mt" {
					t.Errorf("gearbox = %q, want mt", in.Gearbox)
				}
			},
		},
		{
			name: "year bound overrides only the bound that is present",
			q:    "پژو مدل ۹۵",
			raw:  "year_max=1400",
			check: func(t *testing.T, in Intent) {
				if in.YearMin != 1395 || in.YearMax != 1400 {
					t.Errorf("years = %d..%d, want 1395..1400", in.YearMin, in.YearMax)
				}
			},
		},
		{
			name: "sources split on commas and drop empties",
			q:    "",
			raw:  "sources=divar,%20bama,",
			check: func(t *testing.T, in Intent) {
				if !reflect.DeepEqual(in.Sources, []string{"divar", "bama"}) {
					t.Errorf("sources = %v, want [divar bama]", in.Sources)
				}
			},
		},
		{
			name: "empty sources clears the source filter",
			q:    "",
			raw:  "sources=",
			check: func(t *testing.T, in Intent) {
				if in.Sources != nil {
					t.Errorf("sources = %v, want nil", in.Sources)
				}
			},
		},
		{
			name: "multi_only and unflagged engage on 1 only",
			q:    "",
			raw:  "multi_only=1&unflagged=0",
			check: func(t *testing.T, in Intent) {
				if !in.MultiOnly || in.Unflagged {
					t.Errorf("multi_only=%v unflagged=%v, want true/false", in.MultiOnly, in.Unflagged)
				}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := Override(ParseQuery(c.q, testVocab()), parseRaw(t, c.raw), testVocab())
			c.check(t, in)
		})
	}
}

func TestNoParametersLeaveTheParseUntouched(t *testing.T) {
	// The regression pin for every pre-override URL in the wild: with nothing
	// present, Override must be a byte-for-byte no-op on the parsed intent.
	parsed := ParseQuery("ارزون‌ترین پژو ۲۰۶ بالای مدل ۹۵ زیر ۵۰۰ میلیون", testVocab())
	merged := Override(parsed, url.Values{}, testVocab())
	if !reflect.DeepEqual(parsed, merged) {
		t.Errorf("override without parameters changed the intent:\n got %+v\nwant %+v", merged, parsed)
	}
}

func keys(res Result) []string {
	out := make([]string, 0, len(res.Specs))
	for _, s := range res.Specs {
		out = append(out, s.Key)
	}
	sort.Strings(out)
	return out
}

func flagged(source string, n int) index.Offer {
	o := index.Offer{Source: source}
	for i := 0; i < n; i++ {
		o.Flags = append(o.Flags, index.Flag{Code: "x"})
	}
	return o
}

func TestOverriddenIntentFiltersAndReports(t *testing.T) {
	cheap := spec("cheap", "peugeot", "206", 1396, 600*million, 1, 1, 0)
	dear := spec("dear", "peugeot", "206", 1396, 800*million, 1, 1, 0)

	multi := spec("multi", "peugeot", "206", 1396, 600*million, 2, 2, 0)
	multi.Offers = []index.Offer{flagged("divar", 0), flagged("bama", 0)}
	single := spec("single", "peugeot", "206", 1396, 600*million, 1, 1, 0)
	single.Offers = []index.Offer{flagged("bama", 0)}

	// "clean" has a flagged sibling; "dirty" is fully flagged, and one offer
	// carries two flags so flag_count == 3 > offer_count == 2, the case that
	// breaks any count-based shortcut.
	clean := spec("clean", "peugeot", "206", 1396, 600*million, 1, 2, 1)
	clean.Offers = []index.Offer{flagged("divar", 0), flagged("divar", 1)}
	dirty := spec("dirty", "peugeot", "206", 1396, 600*million, 1, 2, 3)
	dirty.Offers = []index.Offer{flagged("divar", 2), flagged("divar", 1)}

	cases := []struct {
		name  string
		q     string
		raw   string
		specs []index.Spec
		want  []string
		check func(t *testing.T, res Result)
	}{
		{
			name:  "override raises the ceiling the text set and reports it",
			q:     "پژو زیر ۵۰۰ میلیون",
			raw:   "price_max=700000000",
			specs: []index.Spec{cheap, dear},
			want:  []string{"cheap"},
			check: func(t *testing.T, res Result) {
				if res.Intent.PriceMax != 700_000_000 {
					t.Errorf("intent.price_max = %d, want the merged 700000000", res.Intent.PriceMax)
				}
			},
		},
		{
			name:  "cleared ceiling filters nothing and reports none",
			q:     "پژو زیر ۵۰۰ میلیون",
			raw:   "price_max=",
			specs: []index.Spec{cheap, dear},
			want:  []string{"cheap", "dear"},
			check: func(t *testing.T, res Result) {
				if res.Intent.PriceMax != 0 {
					t.Errorf("intent.price_max = %d, want 0", res.Intent.PriceMax)
				}
			},
		},
		{
			name:  "absent parameters behave exactly as the text alone",
			q:     "پژو زیر ۵۰۰ میلیون",
			raw:   "",
			specs: []index.Spec{cheap, dear},
			want:  []string{},
		},
		{
			name:  "multi_only keeps corroborated specs only",
			q:     "",
			raw:   "multi_only=1",
			specs: []index.Spec{multi, single},
			want:  []string{"multi"},
		},
		{
			name:  "unflagged drops specs whose every offer is flagged",
			q:     "",
			raw:   "unflagged=1",
			specs: []index.Spec{clean, dirty},
			want:  []string{"clean"},
		},
		{
			name:  "sources keeps specs with a matching offer, list intact",
			q:     "",
			raw:   "sources=divar",
			specs: []index.Spec{multi, single},
			want:  []string{"multi"},
			check: func(t *testing.T, res Result) {
				if len(res.Specs[0].Offers) != 2 {
					t.Errorf("offers = %d, want 2: the card must keep showing the whole market",
						len(res.Specs[0].Offers))
				}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := Override(ParseQuery(c.q, testVocab()), parseRaw(t, c.raw), testVocab())
			res := Run(&index.Index{Specs: c.specs}, in, PriorityRelevant, 10)
			if got := keys(res); !reflect.DeepEqual(got, c.want) {
				t.Errorf("matched %v, want %v", got, c.want)
			}
			if c.check != nil {
				c.check(t, res)
			}
		})
	}
}
