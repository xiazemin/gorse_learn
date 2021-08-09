package main

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/PuerkitoBio/goquery"
	colly "github.com/gocolly/colly/v2"
)

func main() {
	c := colly.NewCollector(
		// colly.Async(false),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36"),
		// colly.Debugger(&debug.LogDebugger{}),
	)
	c.SetRequestTimeout(600 * time.Second)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       0 * time.Second,
		RandomDelay: 1 * time.Second,
	})
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	// Find and visit all links
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		e.Request.Visit(e.Attr("href"))
	})
	c.OnHTML("article", func(e *colly.HTMLElement) {
		if e.Attr("class") == "h3 lh-condensed" {
			fmt.Println(e.ChildAttr("a", "href"))
		}
		//fmt.Println("content:",//e.Name, e.Text, e.ChildTexts("href"))
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println(r.StatusCode) //, "context:", string(r.Body))
		// goquery直接读取resp.Body的内容
		htmlDoc, err := goquery.NewDocumentFromReader(bytes.NewReader(r.Body))

		// 读取url再传给goquery，访问url读取内容，此处不建议使用
		// htmlDoc, err := goquery.NewDocument(resp.Request.URL.String())

		if err != nil {
			log.Fatal(err)
		}

		// 找到抓取项 <div class="hotnews" alog-group="focustop-hotnews"> 下所有的a解析
		//article h1 a
		//.project-title .title
		htmlDoc.Find(".project-title .title").Each(func(i int, s *goquery.Selection) {
			band, _ := s.Attr("href")
			title := s.Text()
			fmt.Printf("项目 %d: %s - %s\n", i, title, band)
			c.Visit(r.Request.URL.Scheme + "://" + r.Request.URL.Host + band)
		})
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Something went wrong:", err)
		c.Visit(r.Request.URL.RawQuery)
	})

	c.OnScraped(func(r *colly.Response) {
		fmt.Println("Finished", r.Request.URL)
	})
	//https://github.com/trending
	//https://gitee.com/explore/all
	c.Visit("https://gitee.com/explore/all")
}
