package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

var visitedURL = make(map[string]bool)
var visitedURLList = []string{}

type visitData struct {
	url           string
	targetElement string
}

type uniName struct {
	main  string
	alias []string
}

func crawlNodeAction(n *html.Node, pre, post func(n *html.Node)) {
	//copy from other website
	//TODO: Why?
	if pre != nil {
		// 將節點傳入閉包函式執行
		pre(n)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		// 遞迴
		crawlNodeAction(c, pre, post)
	}

	if post != nil {
		// 將節點傳入閉包函式執行
		post(n)
	}
}

func checkHTTPOK(res *http.Response, err error) bool {

	if err != nil {
		fmt.Println(err)
		return false
	}

	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		fmt.Println(res.StatusCode)
		return false
	}

	return true
}

func parseHTML(res *http.Response) *html.Node {
	doc, err := html.Parse(res.Body)
	res.Body.Close()

	if err != nil {
		fmt.Println("Error: %v", err)
		return nil
	}

	return doc
}

func crawlUniName(data visitData) ([]string, error) {

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, _ := http.NewRequest("GET", data.url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0 Safari/537.36")

	res, err := client.Do(req)

	if !checkHTTPOK(res, err) {
		res.Body.Close()
		return nil, fmt.Errorf("getting %s: %s", data.url, res.Status)
	}

	doc := parseHTML(res)
	res.Body.Close()

	uniNames := []string{}

	appendNodeText := func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				uniNames = append(uniNames, c.Data)
			}
		}
	}

	var parentNode *html.Node = nil

	findUniName := func(n *html.Node) {
		// 找 <b>
		if n.Type == html.ElementNode && n.Data == data.targetElement {
			// 確認 b 的 Parent 就是 p
			if n.Parent != nil && n.Parent.Type == html.ElementNode && n.Parent.Data == "p" {

				if parentNode == nil || parentNode == n.Parent {
					parentNode = n.Parent
					appendNodeText(n)
				}
			}
		}
	}
	crawlNodeAction(doc, findUniName, nil)

	return uniNames, nil
}

func conCrawlUniName(urls <-chan string, uni_names chan<- []string, wg *sync.WaitGroup) {

	defer wg.Done()
	data := visitData{}
	data.targetElement = "b"
	for url := range urls {
		//start working!!
		data.url = url
		names, _ := crawlUniName(data)

		uni_names <- names

	}
}

func crawlSchURL(data visitData) ([]string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, _ := http.NewRequest("GET", data.url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0 Safari/537.36")

	res, err := client.Do(req)

	if !checkHTTPOK(res, err) {
		res.Body.Close()
		return nil, fmt.Errorf("getting %s: %s", data.url, res.Status)
	}

	doc := parseHTML(res)
	res.Body.Close()

	allUniLinks := []string{}

	findLinks := func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key != "href" {
					continue
				}

				link, err := res.Request.URL.Parse(a.Val)

				if err != nil {
					continue
				}

				if _, exist := visitedURL[link.String()]; exist {
					visitedURLList = append(visitedURLList, link.String())
					continue
				}
				visitedURL[link.String()] = true

				allUniLinks = append(allUniLinks, link.String())
			}
		}
	}

	findDiv := func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == data.targetElement {

			for _, attr := range n.Attr {
				if attr.Key == "class" {

					if attr.Val == "div-col" {
						crawlNodeAction(n, findLinks, nil)
					}

				}
			}
		}
	}
	crawlNodeAction(doc, findDiv, nil)

	return allUniLinks, nil
}

func saveMapToJSON(filename string, data map[string][]string) error {
	fmt.Printf("正在將 %d 筆 Map 資料寫入 %s ...\n", len(data), filename)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // 美化輸出

	// 直接 Encode map 即可，Go 會自動轉成 JSON Object
	if err := encoder.Encode(data); err != nil {
		return err
	}

	return nil
}

func main() {
	fmt.Println("Hello start main")

	//GO fetch all university URL
	all_uni_url := "https://en.wikipedia.org/wiki/Index_of_colleges_and_universities_in_the_United_States"
	tags_to_find := "div"
	data := visitData{all_uni_url, tags_to_find}
	url_need_crawl, _ := crawlSchURL(data)
	println("total number of urls: %d", len(url_need_crawl))

	//define channel
	urls := make(chan string, 10)                 //max 10 jobs to push
	spiders := new(sync.WaitGroup)                // monitor helper
	uni_names_from_job := make(chan []string, 20) //store at least 20 slice
	nums_worker := 10
	for i := 0; i < nums_worker; i++ { //new worker to let it go
		spiders.Add(1)
		go conCrawlUniName(urls, uni_names_from_job, spiders)
	}

	//assign work
	go func() {
		for _, url := range url_need_crawl { //will block on 10 jobs at the same time
			urls <- url
		}
		close(urls)
		fmt.Println("No jobs need to put in")
	}()

	go func() {
		spiders.Wait()
		//worker finished
		close(uni_names_from_job)
		fmt.Println("Finished workers")
	}()

	uni_names_to_display := make(map[string][]string)

	// ... 啟動監控與收集 ...
	start := time.Now()
	count := 0

	for uni_names := range uni_names_from_job {
		count++
		// 📊 每收集 100 個印一次進度，不要每一個都印 (I/O 很慢)
		if count%100 == 0 {
			fmt.Printf("已完成: %d / 2917 (耗時: %v)\n", count, time.Since(start))
		}

		schoolName := strings.TrimSpace(uni_names[0])
		if schoolName == "Also:" || schoolName == "" {
			continue
		}
		uni_names_to_display[schoolName] = uni_names[1:]
	}
	duration := time.Since(start) // ⏱️ 結束計時

	// for k, v := range uni_names_to_display {
	// 	fmt.Println("name: %s, alias: %v", k, v)
	// }
	fmt.Printf("🎉 全部完成！\n總數: %d\n總耗時: %v\n平均吞吐量: %.2f req/s\n",
		count, duration, float64(count)/duration.Seconds())
	fmt.Println("The total school name: %d", len(uni_names_to_display))

	if err := saveMapToJSON("schools.json", uni_names_to_display); err != nil {
		fmt.Println("存檔失敗:", err)
	} else {
		fmt.Println("存檔成功！")
	}

}
