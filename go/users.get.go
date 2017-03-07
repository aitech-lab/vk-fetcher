package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
)

// fetcher
func fetcher(urls <-chan string, out chan<- string) {

	transport := &http.Transport{}
	client := &http.Client{Transport: transport}

	// close all at the end
	defer transport.CloseIdleConnections()
	// defer close(out)

	req, _ := http.NewRequest("GET", "", nil)
	req.Header.Add("Connection", "Keep-Alive")
	req.Header.Add("Accept-Encoding", "gzip")

	// defer transport.CancelRequest(req)

	for path := range urls {
		fmt.Println(path)
		req.URL, _ = url.Parse(path)
		res, err := client.Do(req)
		fmt.Println(err, path)
		if err != nil || res.StatusCode != 200 {
			fmt.Println("[ ERR ]", err, res.Status)
			continue
		}

		reader, _ := gzip.NewReader(res.Body)
		data, _ := ioutil.ReadAll(reader)
		reader.Close()
		res.Body.Close()

		out <- string(data)
	}
}

func feeder(lines chan<- string) {
	bio := bufio.NewReader(os.Stdin)
	defer close(lines)
	for {
		line, _, err := bio.ReadLine()
		if err != nil {
			break
		} else {
			lines <- "https://api.vk.com/method/users.get?v=3&user_ids=" + string(line)
		}
	}
}

func feeder_2(lines chan<- string) {
	for i := 0; i < 100; i += 1 {
		lines <- "https://api.vk.com/method/users.get?v=3&user_ids=" + strconv.Itoa(i)
	}
}

const fetchers = 200

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	urls := make(chan string)
	results := make(chan string)

	go fetcher(urls, results)
	go feeder_2(urls)

	for res := range results {
		fmt.Println(res)
	}

}

/*
package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// fetcher
func fetcher(urls <-chan string, out chan<- string) {

	transport := &http.Transport{}
	client := &http.Client{Transport: transport}

	// close all at the end
	defer transport.CloseIdleConnections()
	// defer close(out)

	req, _ := http.NewRequest("GET", "", nil)
	req.Header.Add("Connection", "Keep-Alive")
	req.Header.Add("Accept-Encoding", "gzip")

	// defer transport.CancelRequest(req)

	for path := range urls {
		req.URL, _ = url.Parse(path)
		res, err := client.Do(req)
		fmt.Println(res.Status, path)
		if err != nil || res.StatusCode != 200 {
			fmt.Println("[ ERR ]", err, res.Status)
			continue
		}

		reader, _ := gzip.NewReader(res.Body)
		data, _ := ioutil.ReadAll(reader)
		reader.Close()
		res.Body.Close()

		out <- string(data)
	}
}

func readFileByLine(path string, urls chan<- string) {
	inFile, _ := os.Open(path)
	defer inFile.Close()
	defer close(urls)
	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)

	prefix := "http://webcache.googleusercontent.com/search?q=cache:otzovik.com"
	// prefix := "http://otzovik.com"
	for scanner.Scan() {
		urls <- prefix + strings.Split(scanner.Text(), "\t")[0]
	}
}

func main() {
	// runtime.GOMAXPROCS(runtime.NumCPU())
	urls := make(chan string)
	results := make(chan string)

	go readFileByLine("uids", urls)
	go fetcher(urls, results)

	out_f, _ := os.Create("./output.txt")

	defer out_f.Close()

	for result := range results {
		qqbytes, _ := out_f.WriteString(result)
		fmt.Println("writed %d bytes", bytes)
		out_f.Sync()
	}
}
*/
