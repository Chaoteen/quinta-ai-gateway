package service

import (
	"context"
	"errors"
	"github.com/Chaoteen/quinta-ai-gateway/common"
	"io"
	"net/http"
	"strings"
	"testing"
)

type assistantRoundTrip func(*http.Request) (*http.Response, error)

func (f assistantRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func assistantTestClient(body string, status int) *http.Client {
	return &http.Client{Transport: assistantRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "www.bing.com" || r.URL.Path != "/search" {
			return nil, errors.New("unexpected search destination")
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
}

func TestAssistantRetrievesOnlyPublicDocs(t *testing.T) {
	for _, tc := range []struct{ q, slug string }{{"如何申请发票", "invoice"}, {"如何兑换卡券", "voucher"}, {"API接入 Base URL", "api-access"}, {"退款和扣费", "quota-and-billing"}} {
		sources := RetrieveAssistantDocuments(tc.q)
		found := false
		for _, s := range sources {
			if s.Href == "/docs/"+tc.slug {
				found = true
			}
			if strings.Contains(s.Snippet, "runtime.env") {
				t.Fatal("operational information exposed")
			}
		}
		if !found {
			t.Fatalf("missing document %s", tc.slug)
		}
	}
	result, err := BuildAssistantContext(context.Background(), "../.env", false)
	if err != nil || len(result.Sources) == 0 || result.WebSearched {
		t.Fatalf("invalid document-only result: %v", err)
	}
	if !strings.Contains(result.Context, "本轮没有联网检索") || !strings.Contains(result.Context, "内测") {
		t.Fatal("missing grounding boundaries")
	}
	b, err := common.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "snippet") {
		t.Fatal("snippets should remain inside context, not duplicate the response")
	}
}

func TestAssistantSearchFiltersUnsafeURLsAndDuplicates(t *testing.T) {
	rss := `<rss><channel><item><title>Official &amp; Docs</title><link>https://example.com/docs</link><description>&lt;b&gt;Public summary&lt;/b&gt;</description></item><item><title>Duplicate</title><link>https://example.com/docs</link></item><item><title>Script</title><link>javascript:alert(1)</link></item><item><title>Loopback</title><link>https://127.0.0.1/secret</link></item><item><title>Private host</title><link>https://admin.local/</link></item><item><title>Credentials</title><link>https://user:pass@example.com/</link></item></channel></rss>`
	sources, err := searchAssistantWeb(context.Background(), "a public question", assistantTestClient(rss, 200))
	if err != nil || len(sources) != 1 {
		t.Fatalf("unsafe result filtering failed: %v %v", sources, err)
	}
	if sources[0].Title != "Official & Docs" || sources[0].Snippet != "Public summary" {
		t.Fatalf("invalid text normalization: %v", sources)
	}
}

func TestAssistantSearchFailureAndSecretQuery(t *testing.T) {
	for _, tc := range []struct {
		body   string
		status int
	}{{"not XML", 200}, {"", 503}, {strings.Repeat("x", 256*1024+1), 200}} {
		if _, err := searchAssistantWeb(context.Background(), "public", assistantTestClient(tc.body, tc.status)); err == nil {
			t.Fatal("search failure silently accepted")
		}
	}
	contacted := false
	client := &http.Client{Transport: assistantRoundTrip(func(r *http.Request) (*http.Response, error) { contacted = true; return nil, errors.New("no") })}
	if _, err := searchAssistantWeb(context.Background(), "api_key=private", client); err == nil || contacted {
		t.Fatal("credentials sent to search provider")
	}
	sources, err := searchAssistantWeb(context.Background(), "public", assistantTestClient(`<rss><channel/></rss>`, 200))
	if err != nil || len(sources) != 0 {
		t.Fatal("empty results must stay empty")
	}
}
