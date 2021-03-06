package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

var wg sync.WaitGroup

type articleText struct {
	Paragraph []string `json:"paragraphs"`
}

type siteMapList struct {
	URL []string `xml:"sitemap>loc"`
}

type newsArticleList struct {
	Articles []newsArticle `xml:"url" json:"articles"`
	Category string        `json:"category"`
}

func (n *newsArticleList) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(n)
}

type newsArticle struct {
	Title         string      `xml:"news>title" json:"title"`
	DatePublished string      `xml:"news>publication_date" json:"datePublished"`
	Keywords      string      `xml:"news>keywords" json:"keywords"`
	ArticleURL    string      `xml:"loc" json:"source"`
	ArticleText   articleText `json:"body"`
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
	defer resp.Body.Close()
	return bytes
}

func getArticlesFromSiteMap(URL string) newsArticleList {
	var l newsArticleList
	bytes := makeRequest(strings.TrimSpace(URL))
	xml.Unmarshal(bytes, &l)

	category := strings.Split(URL, "/")[4]
	l.Category = strings.Split(category, ".")[0]

	return l
}

func getArticleText(c chan newsArticle, article newsArticle) {
	defer wg.Done()

	client := http.Client{}
	req, _ := http.NewRequest("GET", article.ArticleURL, nil)
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	var articleText articleText

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	doc.Find("p.pb-md").Each(func(i int, s *goquery.Selection) {
		articleText.Paragraph = append(articleText.Paragraph, s.Text())
	})

	article.ArticleText = articleText

	c <- article
}

func newsHandler(data newsArticleList) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		Chan := make(chan newsArticle, len(data.Articles))
		for i := range data.Articles {
			wg.Add(1)
			go getArticleText(Chan, data.Articles[i])
		}
		wg.Wait()
		close(Chan)

		var news newsArticleList
		news.Category = strings.Title(data.Category)
		for n := range Chan {
			// TODO: Convert date/time to a more readable format
			news.Articles = append(news.Articles, n)
		}

		template, err := template.ParseFiles("newsTemplate.html")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(err)
			return
		}
		template.Execute(w, news)
		//fmt.Println(news.ToJSON(os.Stdout))
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: indexHandler should show the 2 latest news articles from each category
	var s siteMapList
	bytes := makeRequest("https://www.washingtonpost.com/news-sitemaps/index.xml")
	xml.Unmarshal(bytes, &s)

	var data newsArticleList
	for i := 0; i < (len(s.URL) - 1); i++ {
		go getArticlesFromSiteMap(s.URL[i])
	}

	template, err := template.ParseFiles("indexTemplate.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
	}
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

	// TODO: We shouldn't get the news for a category until someone navigates to the page...
	// TODO: ...otherwise, the articles could be old and user won't see anything recent.
	for c := range categoryMap {
		http.HandleFunc(("/" + c), newsHandler(categoryMap[c]))
	}

	// TODO: Handle this like the other routes.
	// Pass in categories to create links to other pages by category.
	http.HandleFunc("/", indexHandler)
	http.ListenAndServe(":8000", nil)
}
