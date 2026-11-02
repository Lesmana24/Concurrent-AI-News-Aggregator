package crawler

import (
	"encoding/xml"
	"html"
	"regexp"
	"strings"
	"time"
)

type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title string    `xml:"title"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

type atomRoot struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string     `xml:"title"`
	Links     []atomLink `xml:"link"`
	Summary   string     `xml:"summary"`
	Content   string     `xml:"content"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

var (
	htmlTagRegex = regexp.MustCompile(`<[^>]*>`)
	spaceRegex   = regexp.MustCompile(`\s+`)
)

// ParseFeed parses RSS 2.0 or Atom XML bytes into unified FeedItems.
func ParseFeed(data []byte, defaultSource, defaultCategory string) ([]FeedItem, error) {
	// 1. Try parsing as RSS 2.0
	var rss rssRoot
	if err := xml.Unmarshal(data, &rss); err == nil && len(rss.Channel.Items) > 0 {
		sourceName := defaultSource
		if sourceName == "" && strings.TrimSpace(rss.Channel.Title) != "" {
			sourceName = strings.TrimSpace(rss.Channel.Title)
		}

		items := make([]FeedItem, 0, len(rss.Channel.Items))
		for _, it := range rss.Channel.Items {
			link := strings.TrimSpace(it.Link)
			if link == "" && strings.HasPrefix(strings.TrimSpace(it.GUID), "http") {
				link = strings.TrimSpace(it.GUID)
			}
			if link == "" {
				continue
			}

			title := cleanHTML(it.Title)
			if title == "" {
				continue
			}

			desc := cleanHTML(it.Description)
			pubDate := parseFeedDate(it.PubDate)

			items = append(items, FeedItem{
				Title:       title,
				URL:         link,
				Description: desc,
				PublishedAt: pubDate,
				Source:      sourceName,
				Category:    defaultCategory,
			})
		}
		return items, nil
	}

	// 2. Try parsing as Atom
	var atom atomRoot
	if err := xml.Unmarshal(data, &atom); err == nil && len(atom.Entries) > 0 {
		sourceName := defaultSource
		if sourceName == "" && strings.TrimSpace(atom.Title) != "" {
			sourceName = strings.TrimSpace(atom.Title)
		}

		items := make([]FeedItem, 0, len(atom.Entries))
		for _, ent := range atom.Entries {
			var link string
			for _, l := range ent.Links {
				if l.Rel == "" || l.Rel == "alternate" {
					link = strings.TrimSpace(l.Href)
					break
				}
			}
			if link == "" && len(ent.Links) > 0 {
				link = strings.TrimSpace(ent.Links[0].Href)
			}
			if link == "" {
				continue
			}

			title := cleanHTML(ent.Title)
			if title == "" {
				continue
			}

			desc := cleanHTML(ent.Summary)
			if desc == "" {
				desc = cleanHTML(ent.Content)
			}

			dateStr := ent.Published
			if dateStr == "" {
				dateStr = ent.Updated
			}
			pubDate := parseFeedDate(dateStr)

			items = append(items, FeedItem{
				Title:       title,
				URL:         link,
				Description: desc,
				PublishedAt: pubDate,
				Source:      sourceName,
				Category:    defaultCategory,
			})
		}
		return items, nil
	}

	return nil, xml.Unmarshal(data, &rss)
}

func cleanHTML(raw string) string {
	raw = html.UnescapeString(raw)
	raw = htmlTagRegex.ReplaceAllString(raw, " ")
	raw = spaceRegex.ReplaceAllString(raw, " ")
	return strings.TrimSpace(raw)
}

func parseFeedDate(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now()
	}

	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05-0700",
		"2006-01-02 15:04:05",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
	}

	for _, fmtStr := range formats {
		if t, err := time.Parse(fmtStr, raw); err == nil {
			return t
		}
	}

	return time.Now()
}

