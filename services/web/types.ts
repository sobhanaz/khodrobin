export interface Flag { code: string; message: string }

export interface Offer {
  source: string; source_fa: string; title: string | null
  price: number; mileage_km: number | null
  colour: string | null; body_status: string | null; city: string | null
  url: string; seen_at: string; flags: Flag[] | null
  vs_median_pct: number
  raw_brand: string | null; raw_model: string | null; raw_trim: string | null
}

export interface Spec {
  key: string
  brand: string; brand_fa: string
  model: string; model_fa: string
  trim: string | null
  gearbox: string; gearbox_fa: string | null
  year: number; km_bucket: number | null
  offer_count: number; source_count: number
  median_price: number; min_price: number; max_price: number
  flag_count: number
  offers: Offer[]
  score: number
  breakdown: Record<string, number>
}

export interface Intent {
  brand?: string; brand_fa?: string
  model?: string; model_fa?: string
  year_min?: number; year_max?: number
  price_min?: number; price_max?: number
  gearbox?: string
  priority: string; source: string; confidence: number
}

export interface SearchResponse {
  query: string
  result: { intent: Intent; mode: string; total: number; specs: Spec[] }
  timing: { parse_ms: number; rank_ms: number; total_ms: number }
}

export interface StatsResponse {
  built_at: string; loaded_at: string; offers: number
  stats: {
    listings: number; indexed: number; unresolved: number; resolved_pct: number
    specs: number; multi_source_specs: number; flagged_offers: number; sources: string[]
  }
}
