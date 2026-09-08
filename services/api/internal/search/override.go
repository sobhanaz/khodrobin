package search

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/sobhanaz/khodrobin/api/internal/index"
)

// Override merges explicit URL filter parameters into a parsed intent.
//
// The rule is per field and mechanical: a parameter present in the URL wins
// over whatever the text parse said, present-but-empty means the user cleared
// a chip the parse had set (so the text's value is suppressed), and an absent
// parameter leaves the parsed value alone. The merged intent is what filters
// and what the response reports, so a shared /search URL renders the same
// state for whoever opens it.
func Override(in Intent, v url.Values, vocab index.Vocab) Intent {
	if v.Has("brand") {
		in.Brand = strings.TrimSpace(v.Get("brand"))
		// The Fa label comes from the vocab, not the parse: after an override
		// the parsed label may name a different brand, and the chip renders
		// the label.
		in.BrandFa = brandFa(vocab, in.Brand)
	}
	if v.Has("model") {
		in.Model = strings.TrimSpace(v.Get("model"))
		in.ModelFa = modelFa(vocab, in.Brand, in.Model)
	}
	if v.Has("gearbox") {
		in.Gearbox = strings.TrimSpace(v.Get("gearbox"))
	}

	overrideNum(v, "price_min", &in.PriceMin)
	overrideNum(v, "price_max", &in.PriceMax)
	overrideNum(v, "year_min", &in.YearMin)
	overrideNum(v, "year_max", &in.YearMax)

	if v.Has("sources") {
		in.Sources = nil
		for _, s := range strings.Split(v.Get("sources"), ",") {
			if s = strings.TrimSpace(s); s != "" {
				in.Sources = append(in.Sources, s)
			}
		}
	}
	if v.Has("multi_only") {
		in.MultiOnly = v.Get("multi_only") == "1"
	}
	if v.Has("unflagged") {
		in.Unflagged = v.Get("unflagged") == "1"
	}
	return in
}

// overrideNum applies one numeric parameter: empty clears, a valid value
// wins. A value that does not parse, or a negative one, is treated as absent
// rather than as a clear, so a hand-mangled URL degrades to the text's own
// filter instead of silently dropping it.
func overrideNum[T int | int64](v url.Values, name string, dst *T) {
	if !v.Has(name) {
		return
	}
	raw := strings.TrimSpace(v.Get(name))
	if raw == "" {
		*dst = 0
		return
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return
	}
	*dst = T(n)
}

func brandFa(vocab index.Vocab, slug string) string {
	for _, b := range vocab.Brands {
		if b.Slug == slug {
			return b.Fa
		}
	}
	return ""
}

// modelFa prefers the model filed under the active brand: model slugs are not
// guaranteed unique across marques.
func modelFa(vocab index.Vocab, brand, slug string) string {
	fallback := ""
	for _, m := range vocab.Models {
		if m.Slug != slug {
			continue
		}
		if m.Brand == brand {
			return m.Fa
		}
		if fallback == "" {
			fallback = m.Fa
		}
	}
	return fallback
}
