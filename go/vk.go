//
// RDB
// cf:{cid}:{feature_name} sorted set
//    feature_value: count
//    feature_value: count
//    _cnt: total count

package vk

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/fzzy/radix/redis"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const vkFeatures = "sex,bdate,city,country,has_mobile,contacts,connections,status,last_seen,relation,relatives,screen_name,maiden_name,occupation,activities,interests,music,movies,tv,books,games,about,quotes,personal"

const base = "vs43.ailove.ru:6789"

type City struct {
	Cid        string
	UsersTotal string
	Uids       []string
}

/// users.get
/// vk method

func UsersGet(uids <-chan string, features []string, fetchersPerCity int) <-chan *map[string]interface{} {

	rdb, _ := redis.Dial("tcp", base)
	defer rdb.Close()

	out := make(chan *map[string]interface{})
	urls := make(chan string)
	features_str := strings.ToLower(strings.Join(features, ","))

	go func() {
		for uid := range uids {
			urls <- fmt.Sprintf("http://api.vk.com/method/users.get?user_ids=%v&v=5.33&fields=%v", uid, features_str)
		}
		close(urls)
	}()

	for i := 0; i < fetchersPerCity; i++ {
		go fetcher(urls, out)
	}

	return out
}

// yog cities

func GetCities(maxPopulation int) <-chan *City {

	cities := make(chan *City)

	go func() {

		fmt.Println("Get sorted cities")

		rdb, _ := redis.Dial("tcp", base)
		cids, err := rdb.Cmd("zrevrangebyscore", "cities_sorted", "+inf", maxPopulation, "withscores").List()
		rdb.Close()

		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Got %v cities\n", len(cids)/2)

		for i := 0; i < len(cids)/2; i++ {
			cities <- &City{cids[i*2], cids[i*2+1], []string{}}
		}
		close(cities)

	}()
	return cities
}

// Get random users

func GetRandCityUsers(ci <-chan *City, count int) <-chan *City {

	out := make(chan *City)

	go func() {

		rdb, _ := redis.Dial("tcp", base)
		for city := range ci {
			// fmt.Printf("Get uids for %v\n", city.cid)
			cu := fmt.Sprintf("cu/%v", city.Cid)
			uids := rdb.Cmd("SRANDMEMBER", cu, count)
			if uids.Err != nil {
				fmt.Println(uids.Err)
				continue
			}
			city.Uids, _ = uids.List()
			// fmt.Println("Got uids, put to chanel")
			out <- city
		}
		rdb.Close()
	}()
	return out
}

func GetUsersFeatures(cities <-chan *City, features []string) <-chan *City {

	out := make(chan *City)

	go func() {

		for city := range cities {

			uids := make(chan string)
			users := UsersGet(uids, features, 100)

			// start feeder
			go func() {
				for _, uid := range city.Uids {
					uids <- uid
				}
				close(uids)
			}()
			out <- statistics(city, users, features)
		}
	}()

	return out
}

func updateFeatureStat(rdb *redis.Client, cid string, feature string, value interface{}) {

	if feature != "bdate" {

		vstr := fmt.Sprintf("%v", value)
		stid := fmt.Sprintf("st:%v:%v", cid, feature)

		res := rdb.Cmd("zincrby", stid, 1, vstr)
		if res.Err != nil {
			fmt.Println(res.Err)
		}
		rdb.Cmd("zincrby", stid, 1, "_count")

	} else {

		var date []string
		var day, month, year string

		switch strings.Count(value.(string), ".") {
		case 1:
			date = strings.Split(value.(string), ".")
			day = date[0]
			month = date[1]
			year = "-"
		case 2:
			date = strings.Split(value.(string), ".")
			day = date[0]
			month = date[1]
			year = date[2]
		}

		stid := fmt.Sprintf("st:%v:%v", cid, "bdate-md")
		rdb.Cmd("zincrby", stid, 1, month+"-"+day)
		rdb.Cmd("zincrby", stid, 1, "_count")

		stid = fmt.Sprintf("st:%v:%v", cid, "bdate-d")
		rdb.Cmd("zincrby", stid, 1, day)
		rdb.Cmd("zincrby", stid, 1, "_count")

		stid = fmt.Sprintf("st:%v:%v", cid, "bdate-m")
		rdb.Cmd("zincrby", stid, 1, month)
		rdb.Cmd("zincrby", stid, 1, "_count")

		stid = fmt.Sprintf("st:%v:%v", cid, "bdate-y")
		rdb.Cmd("zincrby", stid, 1, year)
		rdb.Cmd("zincrby", stid, 1, "_count")

	}

}

// statistics
func statistics(city *City, data <-chan *map[string]interface{}, features []string) *City {

	rdb, _ := redis.Dial("tcp", base)
	defer rdb.Close()

A:
	for i := 0; i < len(city.Uids); i++ {
		var d *map[string]interface{}
		select {
		case d = <-data:
		case <-time.After(5 * time.Second):
			break A
		}
		for _, feature := range features {
			switch v := (*d)[feature].(type) {
			case int:
			case string:
				updateFeatureStat(rdb, city.Cid, feature, v)
				break
			case map[string]interface{}:
				for sf, sv := range v {
					// fmt.Println(city.Cid, sf, sv)
					updateFeatureStat(rdb, city.Cid, sf, sv)
				}
				break
			}
		}
	}
	return city
}

// fetcher
func fetcher(urls <-chan string, out chan<- *map[string]interface{}) {

	tr := &http.Transport{}
	cl := &http.Client{Transport: tr}
	defer tr.CloseIdleConnections()
	req, _ := http.NewRequest("GET", "", nil)
	req.Header.Add("Connection", "Keep-Alive")
	req.Header.Add("Accept-Encoding", "gzip")
	// defer tr.CancelRequest(req)

	for path := range urls {
		// fmt.Println("%v", path)
		req.URL, _ = url.Parse(path)
		res, err := cl.Do(req)
		if err != nil {
			fmt.Println("[ ERR ]", err)
			return
		}
		rdr, _ := gzip.NewReader(res.Body)
		data, _ := ioutil.ReadAll(rdr)
		rdr.Close()
		res.Body.Close()

		jsonData := map[string]interface{}{}
		json.Unmarshal(data, &jsonData)
		resp := jsonData["response"].([]interface{})
		j := resp[0].(map[string]interface{})
		out <- &j
	}
}

func checkFeature(jsonData *map[string]interface{}, city *City, featureName string) {

	// create feature if there is no one

	// switch t := (*jsonData)[featureName].(type) {
	// case int:
	//     value = fmt.Sprintf("%v", (*jsonData)[featureName].(int))
	// case float64:
	//     value = fmt.Sprintf("%v", (*jsonData)[featureName].(float64))
	// case string:
	//     value = (*jsonData)[featureName].(string)
	// default:
	//     fmt.Println("Unknown type", t)
	// }

	// check featuer in th json, increment feature by name
}
