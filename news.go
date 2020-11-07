package main

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"
)

type siteMapList struct {
	URL []string `xml:"sitemap>loc"`
}

type newsArticleList struct {
	Articles []newsArticle `xml:"url"`
	Category string
}

type newsArticle struct {
	Title         string `xml:"news>title"`
	DatePublished string `xml:"news>publication_date"`
	Keywords      string `xml:"news>keywords"`
	ArticleURL    string `xml:"loc"`
	// TODO: Category should be in here instead of newsArticleList
}

func makeRequest(URL string) []byte {
	client := http.Client{}
	req, _ := http.NewRequest("GET", URL, nil)
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

func getArticlesFromSiteMap(URL string) newsArticleList {
	// TODO: Make all of these requests in parallel!

	var l newsArticleList
	bytes := makeRequest(strings.TrimSpace(URL))
	xml.Unmarshal(bytes, &l)

	// Get the article category by parsing the URL
	category := strings.Split(URL, "/")[4]
	l.Category = strings.Split(category, ".")[0]

	/* Uncomment this block to use JSON
	jsonData, _ := json.Marshal(l)
	fmt.Println(string(jsonData))*/

	/*
		for i := range l.Article {
			fmt.Println("Title:", l.Articles[i].Title)
			fmt.Println("Category:", l.Category)
			fmt.Println("Keywords:", l.Articles[i].Keywords)
			fmt.Println("Published:", l.Articles[i].DatePublished)
			fmt.Println("Location:", l.Articles[i].ArticleURL)
		}
	*/

	return l
}

func politicsHandler(data newsArticleList) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		template, err := template.ParseFiles("newsTemplate.html")
		if err != nil {
			fmt.Println(err)
			return
		}
		template.Execute(w, data)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Create separate NewsArticleLists split by category, and only display the articles from that category
	var s siteMapList
	bytes := makeRequest("https://www.washingtonpost.com/news-sitemaps/index.xml")
	xml.Unmarshal(bytes, &s)

	var data newsArticleList
	for i := 0; i < (len(s.URL) - 1); i++ {
		data = getArticlesFromSiteMap(s.URL[i])
	}

	template, _ := template.ParseFiles("newsTemplate.html")
	template.Execute(w, data)
}

func main() {
	var s siteMapList
	bytes := makeRequest("https://www.washingtonpost.com/news-sitemaps/index.xml")
	xml.Unmarshal(bytes, &s)

	categoryMap := make(map[string]newsArticleList)
	for i := 0; i < (len(s.URL) - 1); i++ {
		data := getArticlesFromSiteMap(s.URL[i])
		categoryMap[data.Category] = data
	}
	fmt.Println(categoryMap["politics"])

	http.HandleFunc("/politics", politicsHandler(categoryMap["politics"]))
	http.HandleFunc("/", indexHandler)
	http.ListenAndServe(":8000", nil)
}
