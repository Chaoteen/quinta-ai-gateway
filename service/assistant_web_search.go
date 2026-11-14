package service

import (
	"context"
	"encoding/xml"
	"errors"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var assistantSearchClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return errors.New("search redirects are not allowed")
	},
}
var assistantHTMLTag = regexp.MustCompile(`<[^>]*>`)
var assistantSecret = regexp.MustCompile(`(?i)(sk-[a-z0-9_-]{12,}|bearer\s+[a-z0-9._-]{12,}|-----BEGIN|api[_ -]?key\s*[:=])`)

// No arbitrary URL fetch: only this fixed public RSS search endpoint is requested.
// This is a best-effort beta adapter, not a contracted production search API.
func searchAssistantWeb(ctx context.Context, query string, client *http.Client) ([]AssistantSource, error) {
	if assistantSecret.MatchString(query) {
		return nil, errors.New("remove credentials before using web search")
	}
	query = strings.TrimSpace(query)
	if runes := []rune(query); len(runes) > 256 {
		query = string(runes[:256])
	}
	endpoint := "https://www.bing.com/search?" + url.Values{"q": {query}, "format": {"rss"}, "count": {"5"}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "QuintaGateway/1.0")
	response, err := client.Do(req)
	if err != nil {
		return nil, errors.New("web search is temporarily unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("web search is temporarily unavailable")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 256*1024+1))
	if err != nil || len(data) > 256*1024 {
		return nil, errors.New("invalid search response")
	}
	var feed struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
		} `xml:"channel>item"`
	}
	if err = xml.Unmarshal(data, &feed); err != nil {
		return nil, errors.New("invalid search response")
	}
	sources := make([]AssistantSource, 0, 5)
	seen := map[string]bool{}
	for _, item := range feed.Items {
		u, err := url.Parse(strings.TrimSpace(item.Link))
		if err != nil || !safeAssistantSourceURL(u) || seen[u.String()] {
			continue
		}
		title := strings.TrimSpace(html.UnescapeString(assistantHTMLTag.ReplaceAllString(item.Title, "")))
		if title == "" {
			continue
		}
		seen[u.String()] = true
		sources = append(sources, AssistantSource{Title: truncateAssistantText(title, 160), Href: u.String(), Snippet: truncateAssistantText(html.UnescapeString(assistantHTMLTag.ReplaceAllString(item.Description, "")), 600)})
		if len(sources) == 5 {
			break
		}
	}
	return sources, nil
}

func safeAssistantSourceURL(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	if u.Scheme != "https" || u.User != nil || u.Port() != "" || host == "" || !strings.Contains(host, ".") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".localhost") {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return false
	}
	return true
}
