package util

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

const DateTempate = "Jan 02 2006 - 3:04 PM MST"
const CacheKey = "defaultDate"

var urlMap = map[string]string{
	"playstation": "https://direct.playstation.com/en-us/consoles/console/playstation5-console.3005816",
	"target":      "https://www.target.com/p/playstation-5-console/-/A-81114595?clkid=13fde857N4bab11ebb2d442010a246e33&lnm=115444&afid=Troposphere%20LLC&ref=tgt_adv_xasd0002",
	"amazon":      "https://www.amazon.com/dp/B08FC6MR62?tag=nismain-20&linkCode=ogi&th=1&psc=1",
	"best":        "https://www.bestbuy.com/site/sony-playstation-5-console/6426149.p?skuId=6426149&irclickid=0gBz1swPLxyLWuHxTSQPxVT4UkEyhURJR0Oj2M0&irgwc=1&ref=198&loc=Troposphere%20LLC&acampID=0&mpid=62662",
	"sam":         "https://www.samsclub.com/b/playstation-4/7330129",
	"b&h":         "https://www.bhphotovideo.com/c/buy/sony-ps5/ci/48556",
	"walmart":     "https://www.walmart.com/ip/Sony-PlayStation-5/363472942?irgwc=1&sourceid=imp_xS9XpiwPLxyLTTgwUx0Mo38bUkEyhUTVR0Oj2M0&veh=aff&wmlspartner=imp_62662&clickid=xS9XpiwPLxyLTTgwUx0Mo38bUkEyhUTVR0Oj2M0&sharedid=&ad_id=612734&campaign_id=9383",
	"gamestop":    "https://www.gamestop.com/video-games/playstation-5/consoles/products/playstation-5/11108140.html?utm_source=rakutenls&utm_medium=affiliate&utm_content=NowInStock&utm_campaign=10&utm_kxconfid=tebx5rmj3&cid=afl_10000087&affID=77777&sourceID=AKGBlS8SPlM-hW6_h0slKIJubMj1xEsEtg",
}

//Fetches the proper url from the url map mased on description
func GetURL(description string) string {
	d := strings.ToUpper(description)
	for key, value := range urlMap {
		k := strings.ToUpper(key)
		if strings.Contains(d, k) {
			return value
		}
	}

	return "#"
}

//Fetch gets the contents of a url
func Fetch(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("failed reading url input")
	}
	defer resp.Body.Close()
	//body, err := ioutil.ReadAll(resp.Body)
	scanner := bufio.NewScanner(resp.Body)
	var b strings.Builder
	for scanner.Scan() {
		t := scanner.Text()
		b.WriteString(t)
	}

	return b.String(), err
}

// Hash Lazy function to hash the cron results
func Hash(doc string) string {
	// Dumb af, but it's a cheap way to specific the most generic thing
	// you can :-/
	var v interface{}
	json.Unmarshal([]byte(doc), &v) // NB: You should handle errors :-/
	cdoc, _ := json.Marshal(v)
	sum := sha256.Sum256(cdoc)
	return hex.EncodeToString(sum[0:])
}
