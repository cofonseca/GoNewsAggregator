package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type siteMapList struct {
	URL []string `xml:"sitemap>loc"`
}

type newsArticleList struct {
	Article []newsArticle `xml:"url"`
}

type newsArticle struct {
	Title    string `xml:"news>title"`
	Keywords string `xml:"news>keywords"`
	Location string `xml:"loc"`
}

func makeRequest(url string) []byte {
	client := http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	resp.Body.Close()
	return bytes
}

func main() {

	var s siteMapList
	bytes := makeRequest("https://www.washingtonpost.com/news-sitemaps/index.xml")
	xml.Unmarshal(bytes, &s)

	for i := 0; i < (len(s.URL) - 1); i++ {

		var l newsArticleList
		bytes := makeRequest(strings.TrimSpace(s.URL[i]))
		xml.Unmarshal(bytes, &l)

		for j := range l.Article {
			fmt.Println(j)
			fmt.Println("Title:", l.Article[j].Title)
			fmt.Println("Keywords:", l.Article[j].Keywords)
			fmt.Println("Location:", l.Article[j].Location)
		}

	}

}
