package crawler

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

var badUrl = []string{
	"javascript: void(0);",
	"javascript:void(0)",
	"javascript: void(0)",
}

func isBadUrl(url string) bool {
	for _, u := range badUrl {
		if u == url {
			return true
		}
	}
	return false
}

/*
ALLOWED_DOMAINS (字符串切片)，允许的域名，比如 []string{"segmentfault.com", "zhihu.com"}
CACHE_DIR (string) 缓存目录
DETECT_CHARSET (y/n) 是否检测响应编码
DISABLE_COOKIES (y/n) 禁止 cookies
DISALLOWED_DOMAINS (字符串切片)，禁止的域名，同 ALLOWED_DOMAINS 类型
IGNORE_ROBOTSTXT (y/n) 是否忽略 ROBOTS 协议
MAX_BODY_SIZE (int) 响应最大
MAX_DEPTH (int - 0 means infinite) 访问深度
PARSE_HTTP_ERROR_RESPONSE (y/n) 解析 HTTP 响应错误
USER_AGENT (string)
*/
func GetTrending() {
	c := colly.NewCollector(
		// colly.Async(false),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36"),
		// colly.Debugger(&debug.LogDebugger{}),
		colly.ParseHTTPErrorResponse(),
		colly.AllowedDomains("gitee.com"),
	)
	c.SetRequestTimeout(600 * time.Second)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       0 * time.Second,
		RandomDelay: 0 * time.Second,
	})
	visitedUrl := make(map[string]bool)
	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL)
	})

	// Find and visit all links
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		url := e.Attr("href")
		//href="javascript: void(0);"
		if isBadUrl(url) {
			return
		}
		if _, ok := visitedUrl[url]; !ok {
			e.Request.Visit(url)
		}
	})

	/*
		c.OnHTML("article", func(e *colly.HTMLElement) {
			if e.Attr("class") == "h3 lh-condensed" {
				fmt.Println(e.ChildAttr("a", "href"))
			}
			//fmt.Println("content:",//e.Name, e.Text, e.ChildTexts("href"))
		})
	*/

	c.OnResponse(func(r *colly.Response) {
		log.Println(r.StatusCode) //, "context:", string(r.Body))
		if r.StatusCode != 200 {
			return
		}
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
		findUrl := []string{}
		htmlDoc.Find(".project-title .title").Each(func(i int, s *goquery.Selection) {
			band, _ := s.Attr("href")
			title := s.Text()
			fmt.Printf("项目 %d: %s - %s\n", i, title, band)

			if isBadUrl(band) || (r.Request.URL.Scheme != "https" && r.Request.URL.Scheme != "http") {
				log.Println("Scheme-----------", r.Request.URL.Scheme)
				return
			}
			url := r.Request.URL.Scheme + "://" + r.Request.URL.Host + band
			findUrl = append(findUrl, url)
		})
		log.Println("all urls:->", findUrl)

		for _, url := range findUrl {
			log.Println("visit:->", url)
			c.Visit(url)
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Fatal("Something went wrong:", err)
		c.Visit(r.Request.URL.RawQuery)
	})

	c.OnScraped(func(r *colly.Response) {
		log.Panicln("Finished", r.Request.URL)
	})
	//https://github.com/trending
	//https://gitee.com/explore/all
	c.Visit("https://gitee.com/explore/all")
}

//go test -timeout 30s -run ^TestGetTrending$ gitee/crawler
//https://zhuanlan.zhihu.com/p/76629605

// Not following redirect to search.gitee.com because its not in AllowedDomains
//https://stackoverflow.com/questions/53346761/framework-gocolly-redirect-to-https-doesnt-work
//https://github.com/crawlab-team/crawlab
