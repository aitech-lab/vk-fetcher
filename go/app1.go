package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/fzzy/radix/redis"
	"io/ioutil"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const base = "vs43.ailove.ru:6789"
const fetcherCounter = 5000
const usersPerCity = 10000
const maxPopulation =10000
const minPopulation = 1000



type City struct {
	Cid   string
	Users []*User
	Same  float64
}

type User struct {
	Uid     string
	Url     string
	Friends []Friend
	Same    float64
}

type Friend struct {
	City uint64
}

func GetCities() <-chan *City {

	out := make(chan *City)

	go func() {

		fmt.Println("Get sorted cities")

		rdb, _ := redis.Dial("tcp", base)
		defer rdb.Close()
		cids, _ := rdb.Cmd("zrevrangebyscore", "cities_sorted", maxPopulation, minPopulation ).List()

		fmt.Printf("Got %v cities\n", len(cids))

		for _, cid := range cids {
			out <- &City{cid, []*User{}, 0.0}
		}
		close(out)

	}()
	return out
}

func GetRandCityUsers(ci <-chan *City, count int) <-chan *City {

	out := make(chan *City)

	go func() {

		rdb, _ := redis.Dial("tcp", base)
		defer rdb.Close()

		for city := range ci {
			fmt.Printf("Get uids for %v\n", city.Cid)
			cu := fmt.Sprintf("cu/%v", city.Cid)
			uids := rdb.Cmd("SRANDMEMBER", cu, count)
			if uids.Err != nil {
				fmt.Println(uids.Err)
				continue
			}
			city.Users = []*User{}
			users, _ := uids.List()
			for _, uid := range users {
				url := fmt.Sprintf("http://api.vk.com/method/friends.get?user_id=%v&order=random&fields=city", uid)
				city.Users = append(city.Users, &User{uid, url, nil, 0.0})
			}
			out <- city
		}
		close(out)
	}()
	return out
}

var errors uint = 0
var fetched uint = 0

func GetUserFriends(ci <-chan *City) <-chan *City {

	out := make(chan *City, 100)

	// launch huge amount of fetchers with users feed
	users := make(chan *User)
	wg := &sync.WaitGroup{}
	for i := 0; i < fetcherCounter; i++ {
		go fetcher(wg, users)
	}

	// Feed fetchers with *User
	go func() {
		for city := range ci {
			errors = 0
			fetched = 0

			for _, user := range city.Users {
				users <- user
			}

			// wait all fetchers done
			fmt.Println("City ready", city.Cid)
			out <- city
			fmt.Printf("fetched: %v, errors: %v\n", fetched, errors)
		}

		// close fetchers feed
		close(users)
		//wait for every fetcher ends and close out
		go func() {
			wg.Wait()
			close(out)
		}()
	}()

	return out
}

func log(err error) {
	if err != nil {
		fmt.Println("[ERR]", err)
	}
}

// fetcher
func fetcher(wg *sync.WaitGroup, users <-chan *User) {

	//	fmt.Println(wg)
	wg.Add(1)
	defer wg.Done()

	timeout := time.Duration(5 * time.Second)
	tr := &http.Transport{}
	cl := &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}

	defer tr.CloseIdleConnections()

	req, _ := http.NewRequest("GET", "", nil)
	req.Header.Add("Connection", "Keep-Alive")
	req.Header.Add("Accept-Encoding", "gzip")

	for user := range users {
		path := user.Url
		//		fmt.Println("FETCH %v", path)
		var err error

		req.URL, err = url.Parse(path)
		if err != nil {
			//			fmt.Println(" [ ERR ] ", err)
			errors++
			continue
		}

		res, err := cl.Do(req)
		if err != nil {
			//			fmt.Println(" [ ERR ] ", err)
			errors++
			continue
		}

		rdr, err := gzip.NewReader(res.Body)
		if err != nil {
			//			fmt.Println(" [ ERR ] ", err)
			errors++
			continue
		}

		data, err := ioutil.ReadAll(rdr)
		if err != nil {
			//			fmt.Println(" [ ERR ] ", err)
			errors++
			continue
		}

		rdr.Close()
		res.Body.Close()
		fetched++
		jsonData := map[string][]Friend{}
		json.Unmarshal(data, &jsonData)
		if resp, ok := jsonData["response"]; ok {
			user.Friends = resp
		}
	}
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	c := GetCities()
	cu := GetRandCityUsers(c, usersPerCity)
	cuf := GetUserFriends(cu)
	rdb, _ := redis.Dial("tcp", base)
	defer rdb.Close()

	for city := range cuf {

		for _, user := range city.Users {
			if user.Friends != nil && len(user.Friends) >= 50 && len(user.Friends) <= 250 {
				samecity := 0.0
				notsame := 0.0
				for _, friend := range user.Friends {
					//					fmt.Println(city.Cid, "\t", friend.City, "\t")
					switch strconv.FormatUint(friend.City, 10) {
					case "0":
						break
					case city.Cid:
						samecity++
						break
					default:
						notsame++
					}
				}
				user.Same = 100.0 * (samecity / (samecity + notsame))
			}
		}

		counter := 0.0
		for _, user := range city.Users {
			if user.Same > 0.0 {
				city.Same += user.Same
				counter++
			}
		}
		city.Same = city.Same / counter
		fmt.Println(city.Cid, city.Same, "%", counter)
		id := fmt.Sprintf("l1:%v", city.Cid)
		res := rdb.Cmd("set", id, city.Same)
		if res.Err != nil {
			fmt.Println("[err]", res.Err)
		}
	}
}
