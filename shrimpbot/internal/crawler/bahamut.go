package crawler

import (
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var httpClient = &http.Client{ //suggested by AI, it should be used with same client and repeated use
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100, //
	},
}

// Return: Title, Content, Error
func ScrapeBahamut(url string) (string, string, error) {

	if strings.Contains(url, "m.forum.gamer.com.tw") {
		url = strings.Replace(url, "m.forum", "forum", 1)
	}

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0 Safari/537.36")

	resp, err := httpClient.Do(req)

	if resp == nil {
		return "", "", err
	}

	if err != nil {
		return "", "", err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", "", err
	}

	title := doc.Find(".c-post__header__title").First().Text()
	selection := doc.Find(".c-article__content").First()
	content := strings.TrimSpace(selection.Text())

	return title, content, nil
}
