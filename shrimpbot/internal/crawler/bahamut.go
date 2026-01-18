package crawler

import (
	"fmt"
	"log"
	"net/http"
	"os"
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
	log.Println("Scrape URL")
	if strings.Contains(url, "m.forum.gamer.com.tw") {
		url = strings.Replace(url, "m.forum", "forum", 1)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		return "", "", nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0 Safari/537.36")
	c := os.Getenv("BAHA_C_ID_TOKEN_SECRET")
	req.Header.Set("Cookie", c)

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
	titleSelection := doc.Find(".c-post__header__title")
	if titleSelection.Length() == 0 {
		//Blocked by web or invalid website
		return "", "", fmt.Errorf("access denied or invalid url")
	}

	title := doc.Find(".c-post__header__title").First().Text()
	selection := doc.Find(".c-article__content").First()
	selection.Find("br").ReplaceWithHtml("\n")
	selection.Find("img").Each(func(_ int, imgNode *goquery.Selection) {
		src, exists := imgNode.Attr("src")
		if exists {
			imgNode.ReplaceWithHtml(" " + src + " ")
			return
		}
		src, exists = imgNode.Attr("data-src")
		if exists {
			imgNode.ReplaceWithHtml(" " + src + " ")
		}
	})
	content := selection.Text()
	log.Println(content)
	log.Println("Scrape URL Done")

	return title, content, nil
}
