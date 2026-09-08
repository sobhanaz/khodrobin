// Package search turns a free-text Persian query into a filtered, ranked list
// of car specs.
//
// Two things are deliberate here.
//
// First, this is the *deterministic* path. A model may later rewrite a query
// into the same Intent struct, but the parse below always runs and always
// produces a usable answer. If the AI service is slow, broken or unaffordable,
// search still works — it degrades to keyword matching rather than failing.
//
// Second, ranking is a transparent weighted score that returns its own
// breakdown. Every number the UI shows about why a result placed where it did
// comes from Score, not from prose. That is what makes «پشت صحنه» honest.
package search

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/sobhanaz/khodrobin/api/internal/index"
)

// Intent is what we believe the user asked for. Every field is optional: an
// empty Intent means "show me everything", which is a legitimate query.
//
// It is also the response's statement of the *effective* filter state: the
// chips in the UI redraw from it, so after URL overrides are merged in it must
// describe what actually filtered, not what the text said.
type Intent struct {
	Brand      string  `json:"brand,omitempty"`
	BrandFa    string  `json:"brand_fa,omitempty"`
	Model      string  `json:"model,omitempty"`
	ModelFa    string  `json:"model_fa,omitempty"`
	YearMin    int     `json:"year_min,omitempty"`
	YearMax    int     `json:"year_max,omitempty"`
	PriceMax   int64   `json:"price_max,omitempty"`
	PriceMin   int64   `json:"price_min,omitempty"`
	Gearbox    string  `json:"gearbox,omitempty"`
	Priority   string  `json:"priority"`
	Source     string  `json:"source"` // rules | llm
	Confidence float64 `json:"confidence"`

	// The three below only ever come from explicit URL parameters; no text
	// query produces them. They live on Intent anyway so the overridden state
	// flows through the same Matches path as everything else and the response
	// reports them alongside the rest of the filter state.
	Sources   []string `json:"sources,omitempty"`
	MultiOnly bool     `json:"multi_only,omitempty"`
	Unflagged bool     `json:"unflagged,omitempty"`
}

// Priorities are the ranking modes the UI exposes as tabs.
const (
	PriorityRelevant = "relevant"
	PriorityCheapest = "cheapest"
	PriorityLowKm    = "lowkm"
	PriorityValue    = "value"
)

var faDigits = strings.NewReplacer(
	"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4",
	"۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
	"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4",
	"٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
)

var arabicFold = strings.NewReplacer(
	"ي", "ی", "ك", "ک", "ة", "ه", "أ", "ا", "إ", "ا", "آ", "ا", "‌", " ",
)

var (
	nonWord    = regexp.MustCompile(`[،,/\\|_\-–—]+`)
	multiSpace = regexp.MustCompile(`\s+`)
	numberRe   = regexp.MustCompile(`\d+`)
)

// Fold normalizes Persian text for matching. Mirrors extract.fold in the
// Python pipeline; the two must agree or a query will not find its own data.
func Fold(s string) string {
	s = faDigits.Replace(s)
	s = arabicFold.Replace(s)
	s = nonWord.ReplaceAllString(s, " ")
	s = multiSpace.ReplaceAllString(s, " ")
	return strings.ToLower(strings.TrimSpace(s))
}

// Iranian car prices are quoted in shorthand: «زیر ۵۰۰» means 500 million
// tomans, «۲ میلیارد» means 2 billion. A bare number under 10,000 in a price
// context is therefore millions, not tomans.
const (
	million = 1_000_000
	billion = 1_000_000_000
)

// ParseQuery extracts an Intent from free Persian text using only rules.
func ParseQuery(q string, vocab index.Vocab) Intent {
	text := Fold(q)
	in := Intent{Priority: PriorityRelevant, Source: "rules"}
	if text == "" {
		return in
	}

	// Brand, then model within that brand. Longest alias wins so «بی وای دی»
	// is not shadowed by a shorter fragment.
	bestBrand := ""
	for _, b := range vocab.Brands {
		for _, a := range b.Aliases {
			fa := Fold(a)
			if fa != "" && strings.Contains(text, fa) && len(fa) > len(bestBrand) {
				bestBrand, in.Brand, in.BrandFa = fa, b.Slug, b.Fa
			}
		}
	}
	// A model name is usually enough to imply its brand: «۲۰۶ می‌خوام» names no
	// marque but only Peugeot makes a 206. When no brand matched, search every
	// model and adopt the brand of whatever wins.
	searchAllModels := in.Brand == ""
	{
		bestModel := ""
		for _, m := range vocab.Models {
			if !searchAllModels && m.Brand != in.Brand {
				continue
			}
			for _, a := range m.Aliases {
				fa := Fold(a)
				if fa == "" {
					continue
				}
				var hit bool
				if isDigits(fa) {
					hit = containsNumberToken(text, fa)
				} else {
					hit = strings.Contains(text, fa)
				}
				if hit && len(fa) > len(bestModel) {
					bestModel, in.Model, in.ModelFa = fa, m.Slug, m.Fa
					if searchAllModels {
						in.Brand = m.Brand
						for _, b := range vocab.Brands {
							if b.Slug == m.Brand {
								in.BrandFa = b.Fa
							}
						}
					}
				}
			}
		}
	}

	if strings.Contains(text, "اتومات") {
		in.Gearbox = "at"
	} else if strings.Contains(text, "دنده") {
		in.Gearbox = "mt"
	}

	switch {
	case strings.Contains(text, "ارزان") || strings.Contains(text, "ارزون"):
		in.Priority = PriorityCheapest
	case strings.Contains(text, "کم کارکرد") || strings.Contains(text, "کارکرد کم") || strings.Contains(text, "کم کار"):
		in.Priority = PriorityLowKm
	case strings.Contains(text, "بهترین") || strings.Contains(text, "مناسب"):
		in.Priority = PriorityValue
	}

	parsePrice(text, &in)
	parseYear(text, &in)

	// Confidence is what fraction of a useful query we pinned down. It is
	// reported so the UI can say "I understood this much" rather than implying
	// certainty it does not have.
	in.Confidence = 0.3
	if in.Brand != "" {
		in.Confidence = 0.6
	}
	if in.Model != "" {
		in.Confidence = 0.85
	}
	if in.Model != "" && (in.YearMin > 0 || in.PriceMax > 0) {
		in.Confidence = 0.95
	}
	return in
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// containsNumberToken matches a bare number only at a digit boundary, so "206"
// does not match inside "2065000000".
func containsNumberToken(text, num string) bool {
	for _, loc := range numberRe.FindAllStringIndex(text, -1) {
		if text[loc[0]:loc[1]] == num {
			return true
		}
	}
	return false
}

// priceScale converts a shorthand number to tomans. «زیر ۵۰۰» is 500 million;
// «۲ میلیارد» is 2 billion; a fully written 500000000 is already tomans.
//
// The value is a float so «۱.۵ میلیارد» works — parsing it as an integer
// silently truncated to 1 and produced a ceiling of one million tomans, which
// matches no car ever sold.
func priceScale(n float64, unit string) int64 {
	switch {
	case strings.Contains(unit, "میلیارد"):
		return int64(n * billion)
	case strings.Contains(unit, "میلیون"):
		return int64(n * million)
	case n < 10_000:
		return int64(n * million)
	default:
		return int64(n)
	}
}

// Numbers may carry a decimal part: «۱.۵ میلیارد» is 1,500,000,000.
var priceRangeRe = regexp.MustCompile(`بین\s*(\d+(?:\.\d+)?)\s*(?:میلیارد|میلیون)?\s*(?:تا|و)\s*(\d+(?:\.\d+)?)\s*(میلیارد|میلیون)?`)
var priceCeilRe = regexp.MustCompile(`(?:زیر|تا|حداکثر|کمتر از)\s*(\d+(?:\.\d+)?)\s*(میلیارد|میلیون)?`)
var priceFloorRe = regexp.MustCompile(`(?:بالای|بیشتر از|از)\s*(\d+(?:\.\d+)?)\s*(میلیارد|میلیون)`)

// looksLikeAYear reports whether a bare number in a price position is really a
// model year. «بالای ۹۵» is 1395, not 95 million tomans.
func looksLikeAYear(n float64) bool {
	if n != float64(int64(n)) {
		return false // decimals are never years
	}
	i := int64(n)
	return (i >= 60 && i <= 99) || (i >= 1300 && i <= 1450) || (i >= 1990 && i <= 2100)
}

func parsePrice(text string, in *Intent) {
	// «بین ۵۰۰ تا ۸۰۰ میلیون» — a range binds both ends, and its "تا" would
	// otherwise be read as a bare ceiling, so it runs first and wins.
	if m := priceRangeRe.FindStringSubmatch(text); m != nil {
		lo, errLo := strconv.ParseFloat(m[1], 64)
		hi, errHi := strconv.ParseFloat(m[2], 64)
		if errLo == nil && errHi == nil && lo > 0 && hi > 0 && !looksLikeAYear(lo) {
			// A unit written once at the end applies to both numbers.
			in.PriceMin = priceScale(lo, m[3])
			in.PriceMax = priceScale(hi, m[3])
			return
		}
	}
	if m := priceCeilRe.FindStringSubmatch(text); m != nil {
		if n, err := strconv.ParseFloat(m[1], 64); err == nil && n > 0 && !looksLikeAYear(n) {
			in.PriceMax = priceScale(n, m[2])
		}
	}
	if m := priceFloorRe.FindStringSubmatch(text); m != nil {
		if n, err := strconv.ParseFloat(m[1], 64); err == nil && n > 0 {
			in.PriceMin = priceScale(n, m[2])
		}
	}
}

var yearRangeRe = regexp.MustCompile(`مدل\s*(\d{2,4})\s*(?:تا|الی)\s*(\d{2,4})`)
var yearFloorRe = regexp.MustCompile(`(?:بالای|از)\s*(?:مدل\s*)?(\d{2,4})`)
var yearExactRe = regexp.MustCompile(`مدل\s*(\d{2,4})`)

// A four-digit Gregorian year stands on its own: «کیا سراتو ۲۰۱۷» never says
// «مدل», and 2017 cannot be anything else.
var yearBareGregorianRe = regexp.MustCompile(`(?:^|\s)(19\d{2}|20\d{2})(?:\s|$)`)

// normalizeYear turns «۹۶» into 1396 and leaves 1396 alone. Two-digit years
// are how Iranians actually write model years.
func normalizeYear(n int) int {
	switch {
	case n >= 1300 && n <= 1450:
		return n
	case n >= 60 && n <= 99:
		return 1300 + n
	case n >= 0 && n <= 20:
		return 1400 + n
	case n >= 1990 && n <= 2100:
		return n - 621
	}
	return 0
}

func parseYear(text string, in *Intent) {
	// «مدل ۹۰ تا ۹۵» binds both ends and must beat the floor and exact forms.
	if m := yearRangeRe.FindStringSubmatch(text); m != nil {
		lo, errLo := strconv.Atoi(m[1])
		hi, errHi := strconv.Atoi(m[2])
		if errLo == nil && errHi == nil {
			a, b := normalizeYear(lo), normalizeYear(hi)
			if a > 0 && b > 0 && a <= b {
				in.YearMin, in.YearMax = a, b
				return
			}
		}
	}
	if m := yearFloorRe.FindStringSubmatch(text); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			if y := normalizeYear(n); y > 0 {
				in.YearMin = y
				return
			}
		}
	}
	if m := yearExactRe.FindStringSubmatch(text); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			if y := normalizeYear(n); y > 0 {
				in.YearMin, in.YearMax = y, y
				return
			}
		}
	}
	if m := yearBareGregorianRe.FindStringSubmatch(text); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			if y := normalizeYear(n); y > 0 {
				in.YearMin, in.YearMax = y, y
			}
		}
	}
}

// Matches reports whether a spec satisfies the intent's hard filters.
func Matches(s index.Spec, in Intent) bool {
	if in.Brand != "" && s.Brand != in.Brand {
		return false
	}
	if in.Model != "" && s.Model != in.Model {
		return false
	}
	if in.Gearbox != "" && s.Gearbox != in.Gearbox {
		return false
	}
	if in.YearMin > 0 && s.Year < in.YearMin {
		return false
	}
	if in.YearMax > 0 && s.Year > in.YearMax {
		return false
	}
	// Price bounds apply to the cheapest offer: if any offer fits the budget,
	// the spec is worth showing.
	if in.PriceMax > 0 && s.MinPrice > in.PriceMax {
		return false
	}
	if in.PriceMin > 0 && s.MaxPrice < in.PriceMin {
		return false
	}
	// A source filter keeps a spec when any of its offers comes from a chosen
	// marketplace, and deliberately does not rewrite the offer list: the card
	// shows the whole market for that car, not just the sites you filtered by.
	if len(in.Sources) > 0 && !hasAnySource(s, in.Sources) {
		return false
	}
	if in.MultiOnly && s.SourceCount <= 1 {
		return false
	}
	if in.Unflagged && !hasCleanOffer(s) {
		return false
	}
	return true
}

func hasAnySource(s index.Spec, sources []string) bool {
	for _, o := range s.Offers {
		for _, want := range sources {
			if o.Source == want {
				return true
			}
		}
	}
	return false
}

// hasCleanOffer reports whether at least one offer carries no flag. It reads
// the offers rather than comparing FlagCount to OfferCount because flag_count
// counts flags, not flagged offers: one offer can carry two flags and make a
// spec with a clean sibling look fully flagged.
func hasCleanOffer(s index.Spec) bool {
	for _, o := range s.Offers {
		if len(o.Flags) == 0 {
			return true
		}
	}
	return false
}

// Weights are the per-factor multipliers for one ranking mode.
type Weights struct {
	Price       float64 `json:"price"`
	Mileage     float64 `json:"mileage"`
	Age         float64 `json:"age"`
	Corroborate float64 `json:"corroborate"`
	Integrity   float64 `json:"integrity"`
}

// Modes are deliberately few and each one is defensible out loud.
var Modes = map[string]Weights{
	PriorityRelevant: {Price: 0.30, Mileage: 0.20, Age: 0.20, Corroborate: 0.20, Integrity: 0.10},
	PriorityCheapest: {Price: 0.70, Mileage: 0.10, Age: 0.05, Corroborate: 0.10, Integrity: 0.05},
	PriorityLowKm:    {Price: 0.20, Mileage: 0.55, Age: 0.10, Corroborate: 0.10, Integrity: 0.05},
	PriorityValue:    {Price: 0.40, Mileage: 0.20, Age: 0.15, Corroborate: 0.15, Integrity: 0.10},
}

// Scored is a spec with its ranking explained.
type Scored struct {
	index.Spec
	Score     float64            `json:"score"`
	Breakdown map[string]float64 `json:"breakdown"`
}

const currentJalaliYear = 1405

// Score returns a total in [0,1] and the per-factor contributions that produced
// it. The breakdown is not decoration: the UI draws it, so a user can disagree
// with the ranking on the evidence rather than on faith.
func Score(s index.Spec, w Weights, priceFloor, priceCeil int64) (float64, map[string]float64) {
	norm := func(v, lo, hi float64) float64 {
		if hi <= lo {
			return 0.5
		}
		x := (v - lo) / (hi - lo)
		return clamp(x)
	}

	// Cheaper is better, so invert the normalized price.
	price := 1 - norm(float64(s.MedianPrice), float64(priceFloor), float64(priceCeil))

	mileage := 0.5
	if s.KmBucket != nil {
		// Buckets are 25,000 km; 20 buckets is 500,000 km, past which we stop
		// distinguishing.
		mileage = 1 - norm(float64(*s.KmBucket), 0, 20)
	}

	age := norm(float64(s.Year), 1380, currentJalaliYear)

	// A spec corroborated by three marketplaces is a more trustworthy price
	// than one seen once. This is the whole reason for aggregating.
	corroborate := norm(float64(s.SourceCount), 1, 3)

	// Contradictory listings sink. A spec whose offers disagree with themselves
	// should not outrank a clean one at a similar price.
	integrity := 1.0
	if s.OfferCount > 0 {
		integrity = clamp(1 - float64(s.FlagCount)/float64(s.OfferCount))
	}

	b := map[string]float64{
		"price":       w.Price * price,
		"mileage":     w.Mileage * mileage,
		"age":         w.Age * age,
		"corroborate": w.Corroborate * corroborate,
		"integrity":   w.Integrity * integrity,
	}
	total := 0.0
	for _, v := range b {
		total += v
	}
	return total, b
}

func clamp(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// Result is one search response.
type Result struct {
	Intent  Intent   `json:"intent"`
	Mode    string   `json:"mode"`
	Total   int      `json:"total"`
	Specs   []Scored `json:"specs"`
	Weights Weights  `json:"weights"`
}

// Run filters by intent, then ranks. Price normalization uses the *matched*
// set, not the whole index: within a search for 206s, being cheap for a 206 is
// what matters, not being cheap for a car.
func Run(idx *index.Index, in Intent, mode string, limit int) Result {
	w, ok := Modes[mode]
	if !ok {
		mode = in.Priority
		if w, ok = Modes[mode]; !ok {
			mode, w = PriorityRelevant, Modes[PriorityRelevant]
		}
	}

	matched := make([]index.Spec, 0, 64)
	var lo, hi int64
	for _, s := range idx.Specs {
		if !Matches(s, in) {
			continue
		}
		if lo == 0 || s.MedianPrice < lo {
			lo = s.MedianPrice
		}
		if s.MedianPrice > hi {
			hi = s.MedianPrice
		}
		matched = append(matched, s)
	}

	scored := make([]Scored, 0, len(matched))
	for _, s := range matched {
		total, b := Score(s, w, lo, hi)
		scored = append(scored, Scored{Spec: s, Score: total, Breakdown: b})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		return scored[i].Key < scored[j].Key
	})

	total := len(scored)
	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}
	return Result{Intent: in, Mode: mode, Total: total, Specs: scored, Weights: w}
}
