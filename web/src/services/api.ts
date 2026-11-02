import type {
  Article,
  ClusterDetailResponse,
  NewsCompareResponse,
  SemanticSearchResult,
  TrendingClusterResponse,
  UserMe,
} from '../types'

const API_BASE = '/api/v1'
export const DEFAULT_DEMO_KEY = 'test-key-123'

export const api = {
  async getHealth() {
    const res = await fetch(`${API_BASE}/health`)
    return res.json()
  },

  async getMe(apiKey: string): Promise<UserMe> {
    const res = await fetch(`${API_BASE}/user/me`, {
      headers: { 'X-API-KEY': apiKey || DEFAULT_DEMO_KEY },
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `HTTP ${res.status}`)
    }
    return res.json()
  },

  async getTrending(limit = 6): Promise<TrendingClusterResponse> {
    const res = await fetch(`${API_BASE}/news/trending?limit=${limit}`)
    if (!res.ok) throw new Error(`Failed to fetch trending clusters (${res.status})`)
    return res.json()
  },

  async getClusterDetail(id: number): Promise<ClusterDetailResponse> {
    const res = await fetch(`${API_BASE}/news/clusters/${id}`)
    if (!res.ok) throw new Error(`Cluster #${id} not found (${res.status})`)
    return res.json()
  },

  async recomputeClusters(apiKey: string) {
    const res = await fetch(`${API_BASE}/news/clusters/recompute`, {
      method: 'POST',
      headers: { 'X-API-KEY': apiKey || DEFAULT_DEMO_KEY },
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `HTTP ${res.status}`)
    }
    return res.json()
  },

  async getRecentArticles(limit = 20, source = ''): Promise<{ total: number; articles: Article[] }> {
    const query = new URLSearchParams()
    if (limit) query.set('limit', limit.toString())
    if (source) query.set('source', source)
    const res = await fetch(`${API_BASE}/news/recent?${query.toString()}`)
    if (!res.ok) throw new Error(`Failed to load wire articles (${res.status})`)
    return res.json()
  },

  async summarizeURL(apiKey: string, url: string): Promise<Article> {
    const res = await fetch(`${API_BASE}/summarize`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-API-KEY': apiKey || DEFAULT_DEMO_KEY,
      },
      body: JSON.stringify({ url }),
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `Failed to summarize URL (${res.status})`)
    }
    return res.json()
  },

  async compareNews(apiKey: string, urls: string[]): Promise<NewsCompareResponse> {
    const res = await fetch(`${API_BASE}/news/compare`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-API-KEY': apiKey || DEFAULT_DEMO_KEY,
      },
      body: JSON.stringify({ urls }),
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `Failed to analyze media bias (${res.status})`)
    }
    return res.json()
  },

  async semanticSearch(query: string, limit = 6): Promise<{ query: string; results: SemanticSearchResult[] }> {
    const res = await fetch(`${API_BASE}/news/search?q=${encodeURIComponent(query)}&limit=${limit}`)
    if (!res.ok) throw new Error(`Search failed (${res.status})`)
    return res.json()
  },

  async getCrawlerStatus() {
    const res = await fetch(`${API_BASE}/crawler/status`)
    if (!res.ok) throw new Error(`Failed to read harvester status (${res.status})`)
    return res.json()
  },

  async triggerCrawler(apiKey: string) {
    const res = await fetch(`${API_BASE}/crawler/trigger`, {
      method: 'POST',
      headers: { 'X-API-KEY': apiKey || DEFAULT_DEMO_KEY },
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `Crawler trigger failed (${res.status})`)
    }
    return res.json()
  },

  // SSE Real-Time Streaming Consumer
  async streamQA(
    apiKey: string,
    question: string,
    onToken: (token: string) => void,
    onDone: (data: { answer: string; citations: string[]; provider: string }) => void,
    onError: (err: Error) => void,
    signal?: AbortSignal
  ) {
    try {
      const res = await fetch(`${API_BASE}/news/qa/stream`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-API-KEY': apiKey || DEFAULT_DEMO_KEY,
        },
        body: JSON.stringify({ question }),
        signal,
      })

      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err.error || `Streaming failed (${res.status})`)
      }

      if (!res.body) {
        throw new Error('ReadableStream not supported by browser response')
      }

      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { value, done } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        let currentEvent = ''
        for (const line of lines) {
          const cleanLine = line.endsWith('\r') ? line.slice(0, -1) : line
          if (cleanLine === '') {
            currentEvent = ''
            continue
          }

          if (cleanLine.startsWith('event:')) {
            currentEvent = cleanLine.slice(6).trim()
          } else if (cleanLine.startsWith('data:')) {
            const rawData = cleanLine.slice(5)
            if (currentEvent === 'token') {
              onToken(rawData)
            } else if (currentEvent === 'done') {
              try {
                const parsed = JSON.parse(rawData.trim())
                onDone(parsed)
              } catch {
                onDone({ answer: '', citations: [], provider: '' })
              }
            } else if (currentEvent === 'error') {
              throw new Error(rawData.trim())
            }
          }
        }
      }
    } catch (err: unknown) {
      if (err instanceof Error) {
        if (err.name === 'AbortError') return
        onError(err)
      } else {
        onError(new Error(String(err)))
      }
    }
  },
}
