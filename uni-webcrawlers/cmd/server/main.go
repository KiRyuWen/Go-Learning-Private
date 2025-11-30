package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"uni-web-crawler/internal/api"
	"uni-web-crawler/internal/crawler"
	"uni-web-crawler/internal/storage"
)

func runSearchDB(db *sql.DB, keyword string) {
	results, err := storage.SearchSchoolsDB(db, keyword)
	if err != nil {
		log.Fatal(err)
	}

	for idx, school := range results {
		fmt.Printf("%d. %s (Aliases: %v)\n", idx+1, school.Name, school.Aliases)
	}

}

func main() {
	fmt.Println("Hello start main")

	mode := flag.String("mode", "crawl", "operation: 'crawl' or 'search' or 'server' ")
	keyword := flag.String("keyword", "", "keyword only work in search")

	flag.Parse()

	db, err := storage.InitDB()
	if err != nil {
		log.Fatal("Unable start db:", err)
	}
	defer db.Close()

	switch *mode {
	case "crawl":
		crawler.RunCrawler(db)
	case "search":
		if *keyword == "" {
			log.Fatal("No keyword error")
			return
		}
		runSearchDB(db, *keyword)
	case "server":
		api.StartDBServer(db)
	default:
		log.Fatal("No mode error")
	}

}
