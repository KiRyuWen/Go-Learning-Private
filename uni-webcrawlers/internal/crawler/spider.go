package crawler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"uni-web-crawler/internal/storage"

	"golang.org/x/net/html"
)

var visitedURL = make(map[string]bool)
var visitedURLList = []string{}

var httpClient = &http.Client{ //suggested by AI, it should be used with same client and repeated use
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100, //
	},
}

type visitData struct {
	url           string
	targetElement string
	others        map[string]string
}

func crawlNodeAction(n *html.Node, pre, post func(n *html.Node)) {
	//copy from other website
	if pre != nil {
		// pre-order
		pre(n)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		// recur
		crawlNodeAction(c, pre, post)
	}

	if post != nil {
		// post order
		post(n)
	}
}

func checkHTTPOK(res *http.Response, err error) bool {

	if err != nil {
		fmt.Println(err)
		return false
	}

	if res.StatusCode != http.StatusOK {
		fmt.Println(res.StatusCode)
		return false
	}

	return true
}

func parseHTML(res *http.Response) *html.Node {
	doc, err := html.Parse(res.Body)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}

	return doc
}

func crawlUniName(data visitData) ([]string, error) {
	req, _ := http.NewRequest("GET", data.url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0 Safari/537.36")

	res, err := httpClient.Do(req)

	if res == nil {
		return nil, err
	}

	if !checkHTTPOK(res, err) {
		res.Body.Close()
		return nil, fmt.Errorf("getting %s: %s", data.url, res.Status)
	}

	doc := parseHTML(res)
	defer res.Body.Close()

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
		// find <b>
		if n.Type == html.ElementNode && n.Data == data.targetElement {
			// makesure b's parent is <p>
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

func conCrawlUniName(urls <-chan string, uniURLChan chan<- []string, wg *sync.WaitGroup) {

	defer wg.Done()
	data := visitData{}
	data.targetElement = "b"
	for url := range urls {
		//start working!!
		data.url = url
		names, _ := crawlUniName(data)

		uniURLChan <- names

	}
}

func crawlUSUniversityIndexPage(data visitData) ([]string, error) {

	req, _ := http.NewRequest("GET", data.url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0 Safari/537.36")

	res, err := httpClient.Do(req)

	if res == nil {
		return nil, err
	}

	if !checkHTTPOK(res, err) {
		res.Body.Close()
		return nil, fmt.Errorf("getting %s: %s", data.url, res.Status)
	}

	doc := parseHTML(res)
	defer res.Body.Close()

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
	fmt.Printf("Writing Data to %s ...\n", filename)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // indent output

	if err := encoder.Encode(data); err != nil {
		return err
	}

	return nil
}

func RunCrawler(db *sql.DB) {
	//GO fetch all university URL
	indexUniURL := "https://en.wikipedia.org/wiki/Index_of_colleges_and_universities_in_the_United_States"
	targetTag := "div"
	data := visitData{indexUniURL, targetTag, nil}
	uniURLs, _ := crawlUSUniversityIndexPage(data)
	fmt.Printf("total number of urls: %d\n", len(uniURLs))

	//define channel
	urlJobs := make(chan string, 10)      //max 10 jobs to push
	spiders := new(sync.WaitGroup)        // monitor helper
	uniURLChan := make(chan []string, 20) //store at least 20 slice
	numWorkers := 10
	for i := 0; i < numWorkers; i++ { //new worker to let it go
		spiders.Add(1)
		go conCrawlUniName(urlJobs, uniURLChan, spiders)
	}

	//assign work
	go func() {
		for _, url := range uniURLs { //will block on 10 jobs at the same time
			urlJobs <- url
		}
		close(urlJobs)
		fmt.Println("No jobs need to put in")
	}()

	go func() {
		spiders.Wait()
		//worker finished
		close(uniURLChan)
		fmt.Println("Finished workers")
	}()

	uniNames := make(map[string][]string)
	outliners := [][]string{}

	// record time spent
	start := time.Now()
	count := 0

	for names := range uniURLChan {
		count++
		if count%100 == 0 {
			fmt.Printf("Finished: %d / 2917 (Time Spent: %v)\n", count, time.Since(start))
		}

		if len(names) == 0 {
			fmt.Println("Didn't get any name")
			continue
		}

		schoolName := strings.TrimSpace(names[0])
		if schoolName == "Also:" || schoolName == "" {
			outliners = append(outliners, names)
			continue
		}
		uniNames[schoolName] = names[1:]
	}
	duration := time.Since(start)

	fmt.Printf("\nTotal: %d\nTime spent: %v\nAvg throughput: %.2f req/s\n",
		count, duration, float64(count)/duration.Seconds())
	fmt.Printf("The total school name: %d\n", len(uniNames))
	fmt.Printf("The total outline: %d\n", len(outliners))

	if err := saveMapToJSON("schools.json", uniNames); err != nil {
		fmt.Println("Save failed:", err)
	} else {
		fmt.Println("Save Successfully.")
	}

	// if err := InitCreateSchema(db); err != nil {
	// 	log.Fatal("Create Table failed:", err)
	// }

	if err := storage.SaveUniToDB(db, uniNames); err != nil {
		log.Println("Push data into DB failed:", err)
	}
}
