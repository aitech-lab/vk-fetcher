package main

import(
    "os"
    "bufio"
    "fmt"
    "strings"
    "compress/gzip"
    "io/ioutil"
    "net/http"
    "net/url"
    "github.com/garyburd/redigo/redis"
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
    // google crawler
    req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)") 
    req.Header.Add("From", "googlebot@googlebot.com") 
    req.Header.Add("Cookie","GOOGLE_ABUSE_EXEMPTION=ID=1bce302bfc6d2940:TM=1468936645:C=c:IP=195.91.235.66-:S=APGng0tZb36QRakobxKLkPpZP5JlTZ4uPw; NID=82=D72Jm8iIl6Pf13XGwzAMpFeUMyIwG6hz7Mt_LI05Gk3eyzeS3fcso2g8f2DkMiYYisP-5TmPbf5ZwZ-9jMcDtsav6OLArp0WAZTmo3AZKGr4fOkibeAXWhlmOZs3mqiw; refreg=http%3A%2F%2Fipv4.google.com%2Fsorry%2FIndexRedirect%3Fcontinue%3Dhttp%3A%2F%2Fwebcache.googleusercontent.com%2Fsearch%253Fq%253Dcache%3Aotzovik.com%26q%3DCGMSBMNb60IYxeO4vAUiGQDxp4NLwOzV96UkOVsXQbTVnMyNuYnenes; coo=1")
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

func listenRedis(key string, urls chan<- string) {
    rdb, _ := redis.Dial("tcp", base)
    defer rdb.Close()

    for {
        data, err: = rdb.Cmd("brpop", key)
    }
}

func main() {
    // runtime.GOMAXPROCS(runtime.NumCPU())
    urls := make(chan string)
    results := make(chan string)

    go listenRedis("tasks", urls)
    go readFileByLine("../../otzovik.com/data/products.txt", urls)
    go fetcher(urls, results);
    
    out_f, _ := os.Create("./output.txt")

    defer out_f.Close()

    for result := range(results) {
        bytes, _ := out_f.WriteString(result)
        fmt.Println("writed %d bytes", bytes)
        out_f.Sync()
    } 
}
