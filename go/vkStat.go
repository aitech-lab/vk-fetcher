//
// redis cids(top) -> redis uids(cids) -> vk json(uid) -> stat(cid)
//

package main

import (
	"./vk"
	"fmt"
	"runtime"
)

const usersPerCity = 10000
const maxPopulation = 10000

var features = []string{"first_name", "last_name", "sex", "personal", "relation", "bdate"}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())
	cities := vk.GetUsersFeatures(vk.GetRandCityUsers(vk.GetCities(maxPopulation), usersPerCity), features)

	i := 0
	for city := range cities {
		i++
		fmt.Printf("%v: Finshed city %+v\n", i, city.Cid)
	}
}
