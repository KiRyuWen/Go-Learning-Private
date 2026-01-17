package main

import (
	"fmt"
	"log"
	"net/http"
	"testing"
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

func TestNikkeBahamut(t *testing.T) {
	bahaForumNum := 36390
	postNum := 884

	url := fmt.Sprintf("https://forum.gamer.com.tw/C.php?bsn=%d&snA=%d", bahaForumNum, postNum)

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0 Safari/537.36")

	resp, err := httpClient.Do(req)

	if resp == nil {
		log.Fatal(err)
		return
	}

	if err != nil {
		log.Fatal(err)
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.StatusCode)
		log.Fatal(err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
		return
	}

	title := doc.Find(".c-post__header__title").First().Text()

	body := doc.Find(".c-post__body").First().Text()

	t.Logf("Post title: %s", title)
	t.Logf("Body: %s", body)

}
