// Package index loads the pre-built search index produced by the Python
// pipeline (crawler/build_index.py) and holds it in memory.
//
// The index is a build artifact, not a database. It is small (under a megabyte
// for ~1,900 listings), immutable at runtime, and rebuilt by the crawler. That
// makes the read path allocation-free and lock-free, and it means a corrupt
// rebuild cannot take the running service down — the process keeps serving
// whatever it loaded at boot.
package index

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Flag struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Offer struct {
	Source string `json:"source"`
	// The listing's own id at its source, and the other half of the key that
	// addresses /api/v1/details/{source}/{id}. The crawler has emitted it since
	// detail enrichment landed; this struct did not have it, so encoding/json
	// dropped it on load and every detail became unreachable through the API.
	// Seventh time a field has gone missing across this exact boundary, which
	// is why schema_test.go pins the whole shape rather than trusting review.
	SourceID    string  `json:"source_id"`
	SourceFa    string  `json:"source_fa"`
	Title       string  `json:"title"`
	Price       int64   `json:"price"`
	MileageKm   *int    `json:"mileage_km"`
	Colour      *string `json:"colour"`
	BodyStatus  *string `json:"body_status"`
	City        *string `json:"city"`
	URL         string  `json:"url"`
	Image       *string `json:"image"`
	SeenAt      string  `json:"seen_at"`
	Flags       []Flag  `json:"flags"`
	VsMedianPct float64 `json:"vs_median_pct"`
	RawBrand    *string `json:"raw_brand"`
	RawModel    *string `json:"raw_model"`
	RawTrim     *string `json:"raw_trim"`
}

type Spec struct {
	Key         string  `json:"key"`
	Image       *string `json:"image"`
	Brand       string  `json:"brand"`
	BrandFa     string  `json:"brand_fa"`
	Model       string  `json:"model"`
	ModelFa     string  `json:"model_fa"`
	Trim        *string `json:"trim"`
	Gearbox     string  `json:"gearbox"`
	GearboxFa   *string `json:"gearbox_fa"`
	Year        int     `json:"year"`
	KmBucket    *int    `json:"km_bucket"`
	OfferCount  int     `json:"offer_count"`
	SourceCount int     `json:"source_count"`
	MedianPrice int64   `json:"median_price"`
	// False when fewer than three offers back the median. With two, it is the
	// mean of the only two asking prices and no seller is asking it.
	MedianReliable bool    `json:"median_reliable"`
	MinPrice       int64   `json:"min_price"`
	MaxPrice       int64   `json:"max_price"`
	FlagCount      int     `json:"flag_count"`
	Offers         []Offer `json:"offers"`
}

type Stats struct {
	// ListingsCaptured is rows fetched; Listings is what survives collapsing
	// re-crawls of the same ad. Reporting only one of them hides the other.
	ListingsCaptured    int      `json:"listings_captured"`
	DuplicatesCollapsed int      `json:"duplicates_collapsed"`
	Listings            int      `json:"listings"`
	Indexed             int      `json:"indexed"`
	Unresolved          int      `json:"unresolved"`
	ResolvedPct         float64  `json:"resolved_pct"`
	Specs               int      `json:"specs"`
	MultiSourceSpecs    int      `json:"multi_source_specs"`
	FlaggedOffers       int      `json:"flagged_offers"`
	Sources             []string `json:"sources"`
}

type Alias struct {
	Slug    string   `json:"slug"`
	Brand   string   `json:"brand"`
	Fa      string   `json:"fa"`
	Aliases []string `json:"aliases"`
}

type Vocab struct {
	Brands []Alias `json:"brands"`
	Models []Alias `json:"models"`
}

type Index struct {
	BuiltAt string `json:"built_at"`
	Stats   Stats  `json:"stats"`
	Vocab   Vocab  `json:"vocab"`
	Specs   []Spec `json:"specs"`

	// LoadedAt is when this process read the file, which is what the UI shows
	// as data freshness. BuiltAt is when the crawler produced it; the gap
	// between them is how stale the running service is.
	LoadedAt time.Time `json:"-"`
}

// Load reads an index from disk. It fails loudly: an API with no index has
// nothing to say, and starting up healthy while serving zero results would be
// worse than not starting at all.
func Load(path string) (*Index, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read index %s: %w", path, err)
	}
	var idx Index
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, fmt.Errorf("parse index %s: %w", path, err)
	}
	if len(idx.Specs) == 0 {
		return nil, fmt.Errorf("index %s contains no specs", path)
	}
	idx.LoadedAt = time.Now().UTC()
	return &idx, nil
}

// TotalOffers counts offers across every spec.
func (i *Index) TotalOffers() int {
	n := 0
	for _, s := range i.Specs {
		n += len(s.Offers)
	}
	return n
}
