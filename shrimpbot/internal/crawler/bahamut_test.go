package crawler

import (
	"fmt"
	"log"
	"testing"
)

func TestNikkeBahamut(t *testing.T) {
	bahaForumNum := 36390
	postNum := 884

	url := fmt.Sprintf("https://forum.gamer.com.tw/C.php?bsn=%d&snA=%d", bahaForumNum, postNum)

	title, content, err := ScrapeBahamut(url)
	if err != nil {
		log.Fatal(err)
		return
	}

	t.Logf("Post title: %s", title)
	t.Logf("Body: %s", content)

}
