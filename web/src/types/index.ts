export interface Article {
  id?: number
  title: string
  url: string
  summary: string
  source: string
  image_url?: string
  published_at?: string
  created_at?: string
}

export interface TrendingClusterItem {
  id: number
  title: string
  summary: string
  image_url?: string
  secondary_image_url?: string
  article_count: number
  unique_outlets: number
  outlets: string[]
  hotness_score: number
  last_updated: string
}

export interface TrendingClusterResponse {
  total_clusters: number
  computed_at: string
  topics: TrendingClusterItem[]
}

export interface ClusterDetailResponse {
  id: number
  title: string
  summary: string
  image_url?: string
  secondary_image_url?: string
  article_count: number
  unique_outlets: number
  outlets: string[]
  hotness_score: number
  last_updated: string
  articles: {
    title: string
    url: string
    summary: string
    source: string
    image_url?: string
  }[]
}

export interface MediaFraming {
  outlet: string
  url: string
  headline: string
  sentiment: 'Positif' | 'Netral' | 'Negatif' | string
  sentiment_score: number // -1.0 to 1.0
  dominant_frame: string
  selected_facts: string[]
}

export interface NewsCompareResponse {
  topic: string
  consensus_facts: string[]
  media_comparisons: MediaFraming[]
  divergences: string[]
  objective_summary: string
}

export interface SemanticSearchResult {
  id: number
  title: string
  url: string
  source: string
  summary: string
  similarity_score?: number
}

export interface CrawlerStats {
  total_indexed: number
  last_crawled_at: string
  active_feeds: number
  is_running: boolean
}

export interface UserMe {
  id: number
  email: string
  quota: number
  status: string
  api_key: string
}

