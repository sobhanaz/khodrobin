// Package lookup implements the reverse direction of the product.
//
// The search path answers "here is what the market has for this query". This
// path answers "here is where else this exact listing is being sold" — paste a
// URL from any marketplace, and every offer for the same car spec comes back
// with the overpay verdict. It is the same identity resolution search already
// depends on, run in the direction Torob states as their hardest problem but
// nobody demos: listing → same car everywhere, instead of search → results.
//
// The URL never leaves the process. It is parsed, folded to its source and id,
// and answered from the in-memory index — no fetch of the pasted page, no
// dependency on the source site being up, and no request surface for a caller
// to point at anything else.
package lookup

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/sobhanaz/khodrobin/api/internal/index"
)

// ErrUnknown is returned for a URL this product cannot attribute to a source
// it crawls. The handler turns it into a 422 with a Persian message; the
// distinction matters because "not our format" and "not in our index" are
// different user mistakes with different advice.
var ErrUnknown = fmt.Errorf("unknown listing url")

// host → source. Only the hosts the crawler actually visits are listed, and
// www is normalised away so a pasted www.bama.ir URL still resolves.
var hosts = map[string]string{
	"divar.ir":            "divar",
	"bama.ir":             "bama",
	"hamrah-mechanic.com": "hamrah",
	"khodro45.com":        "khodro45",
	"sheypoor.com":        "sheypoor",
}

// Parse reduces a pasted URL to (source, id).
//
// Each rule mirrors where that source's own id actually lives, learned from
// the crawler, not guessed from the URL's shape:
//
//   - divar: the per-listing token is the last path segment of /v/{slug}/{token}
//   - sheypoor: the numeric tail of /v/{slug}-{id}.html — but the slug is an
//     editable title, so only the digits after the final dash are trusted
//   - bama: detail-{id}-… inside /car/
//   - hamrah: the numeric tail of /cars-for-sale/{brand}/{model}/{id}/
//   - khodro45: /used-car/{slug}/, where the slug IS the id
func Parse(raw string) (source, id string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", ErrUnknown
	}
	// Accept a bare pasted URL with no scheme; a user copying from a browser
	// address bar may or may not carry https://.
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", ErrUnknown
	}
	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	src, ok := hosts[host]
	if !ok {
		return "", "", ErrUnknown
	}
	segs := splitPath(u.Path)
	if len(segs) == 0 {
		return "", "", ErrUnknown
	}
	last := segs[len(segs)-1]

	switch src {
	case "divar":
		// /v/{title-slug}/{token} — the token is the LAST segment, so a
		// two-segment path is a slug without its token, not a listing.
		if len(segs) < 3 || segs[0] != "v" {
			return "", "", ErrUnknown
		}
		return src, last, nil
	case "sheypoor":
		// {slug}-{digits}.html — the id is the digit run after the last dash,
		// immune to dashes inside the seller's editable title.
		if len(segs) < 2 || segs[0] != "v" {
			return "", "", ErrUnknown
		}
		last = strings.TrimSuffix(last, ".html")
		if d := tailDigits(last); d != "" {
			return src, d, nil
		}
		return "", "", ErrUnknown
	case "bama":
		if len(segs) < 2 || segs[0] != "car" {
			return "", "", ErrUnknown
		}
		if !strings.HasPrefix(last, "detail-") {
			return "", "", ErrUnknown
		}
		// detail-{id}-{slug…}: the id is the first dash-delimited token.
		if rest := strings.TrimPrefix(last, "detail-"); rest != "" {
			if tok := strings.SplitN(rest, "-", 2)[0]; tok != "" {
				return src, tok, nil
			}
		}
		return "", "", ErrUnknown
	case "hamrah":
		// …/{orderId}/ — unlike Sheypoor there is no slug to strip, the whole
		// final segment IS the id, and it is purely numeric.
		if allDigits(last) {
			return src, last, nil
		}
		return "", "", ErrUnknown
	case "khodro45":
		if len(segs) < 2 || segs[0] != "used-car" {
			return "", "", ErrUnknown
		}
		return src, last, nil
	}
	return "", "", ErrUnknown
}

// splitPath splits a URL path into its non-empty segments.
func splitPath(p string) []string {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	out := parts[:0]
	for _, s := range parts {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// tailDigits returns the digit run after the final dash of s, if any.
func tailDigits(s string) string {
	i := strings.LastIndex(s, "-")
	if i < 0 || i == len(s)-1 {
		return ""
	}
	tail := s[i+1:]
	if allDigits(tail) {
		return tail
	}
	return ""
}

// allDigits reports whether s is one or more ASCII digits and nothing else.
func allDigits(s string) bool {
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

// Result is what the endpoint answers with. The verdict fields are pointers so
// a one-offer spec can say "nothing to compare" honestly instead of reporting
// a zero that would render as «۰٪ زیر میانه».
type Result struct {
	Spec *index.Spec `json:"spec"`
	// The offer the pasted URL belongs to, if it is still in the index.
	Offer *index.Offer `json:"offer,omitempty"`
	// Cheapest clean (unflagged) offer across the spec, nil when every offer
	// is flagged. A flagged price is not a price to act on.
	Cheapest *index.Offer `json:"cheapest_offer,omitempty"`
	// (pasted price − cheapest clean price) as a signed percentage of the
	// cheapest. Positive means the pasted listing costs more.
	OverpayPct *float64 `json:"overpay_pct,omitempty"`
	// How many OTHER marketplaces carry this same spec — the headline number.
	OtherSources int `json:"other_sources"`
	// Why nothing was found, for the honest empty states.
	Note string `json:"note,omitempty"`
}

// Find locates the offer for (source, id) in the index and returns its whole
// spec with the verdict filled in.
func Find(idx *index.Index, source, id string) *Result {
	for i := range idx.Specs {
		spec := &idx.Specs[i]
		var pasted *index.Offer
		for j := range spec.Offers {
			o := &spec.Offers[j]
			if o.Source == source && offerID(o.URL) == id {
				pasted = o
				break
			}
		}
		if pasted == nil {
			continue
		}

		res := &Result{Spec: spec, Offer: pasted}

		// The comparison runs over clean offers only, and the pasted listing
		// itself is excluded: telling someone a listing is 0% more expensive
		// than itself is not a verdict.
		cheapest := int64(0)
		for j := range spec.Offers {
			o := &spec.Offers[j]
			if len(o.Flags) > 0 || o == pasted {
				continue
			}
			if cheapest == 0 || o.Price < cheapest {
				cheapest = o.Price
				res.Cheapest = o
			}
		}
		if res.Cheapest != nil && pasted.Price > res.Cheapest.Price {
			pct := 100 * float64(pasted.Price-res.Cheapest.Price) / float64(res.Cheapest.Price)
			res.OverpayPct = &pct
		}

		seen := map[string]bool{source: true}
		for _, o := range spec.Offers {
			seen[o.Source] = true
		}
		res.OtherSources = len(seen) - 1
		if res.OtherSources == 0 {
			res.Note = "only"
		}
		return res
	}
	return nil
}

// offerID extracts the same id Parse would produce, from a stored offer URL,
// so matching compares id-to-id rather than string-to-string. The crawler's
// source_id comes from the same place, but the index does not carry it — the
// URL is what survives into the index, so the URL is what is re-parsed.
func offerID(raw string) string {
	source, id, err := Parse(raw)
	if err != nil {
		return ""
	}
	_ = source
	return id
}
