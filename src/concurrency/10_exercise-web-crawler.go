package main

import (
	"fmt"
	"sync"
)

// 在这个练习中，我们将会使用 Go 的并发特性来并行化一个 Web 爬虫。
// 修改 Crawl 函数来并行地抓取 URL，并且保证不重复。
// 提示：你可以用一个 map 来缓存已经获取的 URL，但是要注意 map 本身并不是并发安全的！

type Fetcher interface {
	// Fetch 返回 URL 的 body 内容，并且将在这个页面上找到的 URL 放到一个 slice 中。
	Fetch(url string) (body string, urls []string, err error)
}

type SafeMap struct {
	v   map[string]int
	mux sync.Mutex     // 访问互斥锁
	wg  sync.WaitGroup // 等待组
}

var c = SafeMap{v: make(map[string]int)}

func (c *SafeMap) exist(url string) bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	_, ok := c.v[url] // 判断url是否存在
	if ok {
		return true
	} else {
		c.v[url] = 1
		c.wg.Add(1)
		return false
	}
}

// Crawl 使用 fetcher 从某个 URL 开始递归的爬取页面，直到达到最大深度。
func Crawl(url string, depth int, fetcher Fetcher) {
	defer c.wg.Done()

	if depth <= 0 {
		return
	}

	body, urls, err := fetcher.Fetch(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("found: %s %q\n", url, body)
	for _, u := range urls {
		if c.exist(u) == false { // 如果url不存在才递归
			go Crawl(u, depth-1, fetcher) // 并发
		}
	}
	return
}

func main() {
	url := "https://golang.org/"
	c.exist(url)
	Crawl(url, 4, fetcher)
	c.wg.Wait()
}

// fakeFetcher 是返回若干结果的 Fetcher。
type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	if res, ok := f[url]; ok {
		return res.body, res.urls, nil
	}
	return "", nil, fmt.Errorf("not found: %s", url)
}

// fetcher 是填充后的 fakeFetcher。
var fetcher = fakeFetcher{
	"https://golang.org/": &fakeResult{
		"The Go Programming Language",
		[]string{
			"https://golang.org/pkg/",
			"https://golang.org/cmd/",
		},
	},
	"https://golang.org/pkg/": &fakeResult{
		"Packages",
		[]string{
			"https://golang.org/",
			"https://golang.org/cmd/",
			"https://golang.org/pkg/fmt/",
			"https://golang.org/pkg/os/",
		},
	},
	"https://golang.org/pkg/fmt/": &fakeResult{
		"Package fmt",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
	"https://golang.org/pkg/os/": &fakeResult{
		"Package os",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
}
