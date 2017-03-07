package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"time"
)

const fetchers = 800

const max_uid = 10000000
const max_cnt = 800
const max_len = 4000

const fields = "sex,country,city,bdate"

type User struct {
	uid         uint32
	first_name  string
	second_name string
	bdate       string
	sex         int
	city        int
	countyr     int
}

// fetcher
func fetcher(urls <-chan string, out chan<- string) {

	transport := &http.Transport{}
	client := &http.Client{Transport: transport}

	// close all at the end
	defer transport.CloseIdleConnections()
	// defer close(out)

	req, _ := http.NewRequest("GET", "", nil)
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Accept-Encoding", "gzip")

	// defer transport.CancelRequest(req)

	for path := range urls {

		req.URL, _ = url.Parse(path)
		res, err := client.Do(req)

		if err != nil || res.StatusCode != 200 {
			fmt.Println("[ ERR ]", err, res.Status)
			continue
		}

		reader, _ := gzip.NewReader(res.Body)
		data, _ := ioutil.ReadAll(reader)
		reader.Close()
		res.Body.Close()

		jsonData := map[string]interface{}{}
		json.Unmarshal(data, &jsonData)
		resp := jsonData["response"].([]interface{})
		for _, user := range resp {
			u := user.(map[string]interface{})
			line := fmt.Sprint(u["uid"], u["sex"], u["bdate"], u["city"], u["country"])
			out <- line
		}
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
			lines <- "https://api.vk.com/method/users.get?v=3&fields=" + fields + "&user_ids=" + string(line)
		}
	}
}

func feeder_regular(urls chan<- string) {
	defer close(urls)
	uid := 0
	cnt := 0
	uids := ""
	for {
		uid = uid + 1
		cnt = cnt + 1
		if uid > max_uid {
			break
		}
		uids = uids + "," + strconv.Itoa(uid)
		if len(uids) > max_len || cnt > max_cnt {
			urls <- "https://api.vk.com/method/users.get?v=3&fields=" + fields + "&user_ids=" + uids
			uids = ""
			cnt = 0
		}
	}
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())
	urls := make(chan string, 1000)
	results := make(chan string, 1000)
	for i := 0; i < fetchers; i++ {
		go fetcher(urls, results)
	}
	go feeder_regular(urls)

	for {
		select {
		case res := <-results:
			fmt.Println(res)
		case <-time.After(time.Second * 5):
			close(results)
			return
		}
	}

}
