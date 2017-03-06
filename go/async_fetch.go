//home/v.seregin/Tools/go/bin/go run $0 $@ ; exit

package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "runtime"
)

type HttpResponse struct {
    cid      int
    url      string
    response *http.Response
    err      error
    body     map[string]interface{}
}

const count = 100

func fetchVkUser(cid int, co chan<- *HttpResponse) {
    url := fmt.Sprintf("http://api.vk.com/method/users.get?user_ids=%v", cid)
    // fmt.Printf("Fetching %s \n", url)
    var body_json map[string]interface{}
    resp, err := http.Get(url)
    if err != nil {
        fmt.Printf("ERR %s\n", err)
        fmt.Printf("%+v", resp)
    } else {
        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            fmt.Printf("ERR %s\n", err)
            body_json = nil
        }
        resp.Body.Close()
        if err := json.Unmarshal(body, &body_json); err != nil {
            fmt.Printf("ERR %s\n", err)
        }
        // fmt.Println(body_json)
    }
    co <- &HttpResponse{cid, url, resp, err, body_json}
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())
    
    i := 0
    co := make(chan *HttpResponse)

    for i<count {
        i++
        go fetchVkUser(i, co)
    }

    for {
        res := <-co
        if res.err!= nil {
            go fetchVkUser(res.cid, co)
        } else {
            fmt.Println(res.body)
            i++
            go fetchVkUser(i, co)
        }
        if i>=100000 {
            break
        }
        //fmt.Printf("%s status: %s\n", result.url, result.response.Status)
        //fmt.Println(result.body)
    }
}
