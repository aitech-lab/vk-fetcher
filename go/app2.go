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
	//	"strconv"
	"sync"
	"time"
)

const base = "vs43.ailove.ru:6789"
const fetcherCounter = 10000
const usersPerCity = 10000
const maxPopulation = 10000
const minPopulation = 1000

type City struct {
	Cid   string
	Users []*User
	Same  float64
}

type User struct {
	uid     string
	url     string
	wall    *Wall
	private bool
	wg      *sync.WaitGroup
}

type Response struct {
	Response *Wall
	Error    *Error
}

type Wall struct {
	Count uint
	Items []*Item
}

type Item struct {
	Owner_id  uint64
	From_id   uint64
	Post_type string
	Date      int64
	//	Text        string
	Attachments []struct {
		Type string
	}
}

type Error struct {
	Error_code uint
	Error_msg  string
}

func GetCities() <-chan *City {

	out := make(chan *City)

	go func() {

		fmt.Println("Get sorted cities")

		rdb, _ := redis.Dial("tcp", base)
		defer rdb.Close()
		cids, _ := rdb.Cmd("zrevrangebyscore", "cities_sorted", maxPopulation, minPopulation).List()

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
				url := fmt.Sprintf("http://api.vk.com/method/wall.get?owner_id=%v&v=5.34&count=100", uid)
				city.Users = append(city.Users, &User{uid: uid, url: url, wall: nil, private: false, wg: nil})
			}
			out <- city
		}
		close(out)
	}()
	return out
}

var errors uint = 0
var fetched uint = 0

func GetUserWall(ci <-chan *City) <-chan *City {

	out := make(chan *City, 100)

	// launch huge amount of fetchers with users feed
	users := make(chan *User)
	for i := 0; i < fetcherCounter; i++ {
		go fetcher(users)
	}

	// Feed fetchers with *User
	go func() {
		cwg := sync.WaitGroup{}
		for city := range ci {

			cwg.Add(1)

			errors = 0
			fetched = 0

			wg := &sync.WaitGroup{}
			for _, user := range city.Users {
				user.wg = wg
				users <- user
			}

			// wait all fetchers done its fetching
			go func(city *City, wg *sync.WaitGroup) {
				wg.Wait()
				fmt.Println("City ready", city.Cid)
				fmt.Printf("fetched: %v, errors: %v\n", fetched, errors)
				out <- city
				cwg.Done()
			}(city, wg)
		}
		// stop feeding fetchers
		close(users)
		// wait all fetchers done
		cwg.Wait()
		// close out
		close(out)
	}()

	return out
}

// fetcher
func fetcher(users <-chan *User) {

	//	fmt.Println(wg)

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
		user.wg.Add(1)

		path := user.url
		//		fmt.Println("FETCH %v", path)
		var err error

		req.URL, err = url.Parse(path)
		if err != nil {
			//			fmt.Println(" [ ERR ] ", err)
			errors++
			user.wg.Done()
			continue
		}

		res, err := cl.Do(req)
		if err != nil {
			//			fmt.Println(" [ ERR ] ", err)
			errors++
			user.wg.Done()
			continue
		}

		rdr, err := gzip.NewReader(res.Body)
		if err != nil {
			//			fmt.Println(" [ ERR ] ", err)
			errors++
			user.wg.Done()
			continue
		}

		data, err := ioutil.ReadAll(rdr)
		if err != nil {
			//			fmt.Println(" [ ERR ] ", err)
			errors++
			user.wg.Done()
			continue
		}

		rdr.Close()
		res.Body.Close()
		fetched++
		response := Response{}
		err = json.Unmarshal(data, &response)
		if err != nil {
			fmt.Println(err)
			user.wg.Done()
			errors++
			continue
		}

		if response.Error != nil {
			// &{15 Access denied: user hid his wall from accessing from outside}
			//			fmt.Println(response.Error)
			if response.Error.Error_code == 15 {
				user.private = true
			}
		} else {
			resp := response.Response
			user.wall = resp
		}

		user.wg.Done()
	}
}

type Stat struct {
	frequency float64
	atypes    map[string]float64
	repost    float64
	post      float64
}

func extractFrequency(items *[]*Item) *Stat {
	// First post can be pinned so we get second
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("items", items)
			fmt.Println("recovered", r)
		}
	}()
	var frequency float64

	if len(*items) > 50 {
		duration := time.Unix((*items)[1].Date, 0).Sub(time.Unix((*items)[len(*items)-1].Date, 0)).Hours() / 24
		frequency = float64((len(*items) - 1)) / duration
	} else {
		frequency = 0.5
	}
	// normal user activity ~ 0.5 post per day. Greater 2 seems like bot activity
	if frequency > 5 {
		frequency = 0.5
	}
	//	 anomaly check
	//	if frequency > 1 && frequency < 5 {
	//		fmt.Println(duration, "days; ", frequency, "posts/day")
	//		for _, item := range *items {
	//			fmt.Println(time.Unix(item.Date, 0), item.Text, item.Attachments)
	//		}
	//	}

	//	ptypes := &map[string]float64{}
	atypes := map[string]float64{"photo": 0, "photos_list": 0, "audio": 0, "video": 0, "link": 0, "doc": 0}
	var repost float64
	for _, item := range *items {
		//		(*ptypes)[item.Post_type] += 1
		if item.Owner_id != item.From_id {
			repost += 1
		}
		for _, a := range item.Attachments {
			atypes[a.Type] += 1
		}
	}
	//	fmt.Println("\t", atypes, repost)
	return &Stat{frequency, atypes, repost, 0.0}

}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	c := GetCities()
	cu := GetRandCityUsers(c, usersPerCity)
	cities := GetUserWall(cu)

	rdb, _ := redis.Dial("tcp", base)
	defer rdb.Close()

	for city := range cities {

		stat := []*Stat{}
		private := 0.0
		counter := 0.0
		walls := 0
		for _, user := range city.Users {
			counter++
			if user.private {
				private++
			}

			if user.wall != nil && user.wall.Count >= 100 && len(user.wall.Items) >= 100 {
				//	fmt.Println(user.wall.Count)
				ustat := extractFrequency(&user.wall.Items)
				ustat.post = float64(user.wall.Count)
				stat = append(stat, ustat)
				walls++
			}
		}

		if walls < 100 {
			//			for _, user := range city.Users {
			//				if user.wall != nil {
			//					fmt.Println("User wall", user.wall.Count, len(user.wall.Items))
			//				} else {
			//					fmt.Println("USER WALL is NIL", user.wall)
			//				}
			//			}
			fmt.Println("CITY", city.Cid, "HAS NO QUORUM", walls, "/", counter)
		} else {
			avrState := Stat{repost: 0, atypes: map[string]float64{}, frequency: 0, post: 0}
			for _, s := range stat {
				avrState.repost += s.repost
				avrState.post += s.post
				avrState.frequency += s.frequency
				for e, v := range s.atypes {
					avrState.atypes[e] += v
				}
			}

			l := float64(len(stat))
			avrState.repost = avrState.repost / l
			avrState.post = avrState.post / l
			avrState.frequency = avrState.frequency / l
			for e, v := range avrState.atypes {
				avrState.atypes[e] = v / l
			}

			fmt.Println("cid", city.Cid, "count", l, avrState)

			id := fmt.Sprintf("l2:%v", city.Cid)
			rdb.Cmd("set", id, avrState.frequency)

			id = fmt.Sprintf("l3:%v", city.Cid)
			rdb.Cmd("set", id, avrState.atypes["photo"])

			id = fmt.Sprintf("l4:%v", city.Cid)
			rdb.Cmd("set", id, avrState.repost)

			id = fmt.Sprintf("l5:%v", city.Cid)
			rdb.Cmd("set", id, 100.0*private/(private+counter))
			fmt.Println(100.0 * private / (private + counter))
		}
	}

}
