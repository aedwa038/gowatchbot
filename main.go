package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"

	psdb "github.com/aedwa038/gowatcherbot/mysql"
	"github.com/aedwa038/gowatcherbot/scraper"
	sc "github.com/aedwa038/gowatcherbot/slack"
	"github.com/aedwa038/gowatcherbot/util"
	_ "github.com/go-sql-driver/mysql"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

var indexTmpl = template.Must(template.ParseFiles("index.html"))
var signingSecret = mustGetenv("SLACK_SIGNING_KEY")
var api = sc.NewSlackClient(mustGetenv("SLACK_TOKEN"))
var url = mustGetenv("URL")
var slack_channel = mustGetenv("DEFAULT_SLACK_CHANNEL")

// eventHandler endpoints function that handles slack events.
// We use this endpoint for slack verifcations and bot mentions
func eventHandler(w http.ResponseWriter, r *http.Request) {

	body, err := sc.GetRequestBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := sc.ValidateRequestBody(r.Header, body, signingSecret); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	eventsAPIEvent, err := sc.GetEvents(body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ch, err := sc.HandleVerifcationRequest(eventsAPIEvent, body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if ch != "" {
		w.Header().Set("Content-Type", "text")
		w.Write([]byte(ch))
	}

	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			api.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
		}
	}

	//set content-type to json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// handleIndex enpoint for  the app's homepage
func handleIndex(w http.ResponseWriter, r *http.Request) {
	if err := indexTmpl.Execute(w, nil); err != nil {
		log.Printf("Error executing indexTmpl template: %s", err)
	}
}

func getDate(db *sql.DB) (*time.Time, error) {
	var date time.Time
	var err error
	if dateValue, err := psdb.GetConfig(db, util.CacheKey); err == nil {
		date, err = time.Parse(util.DateTempate, dateValue)
	}
	return &date, err
}

// crawl endpoint intended to scrape a site  called throuhg appengines cron config.
func crawl(w http.ResponseWriter, r *http.Request) {

	dbUser := mustGetenv("DB_USER") // e.g. 'my-db-user'
	dbPwd := mustGetenv("DB_PASS")  // e.g. 'my-db-password'
	//dbTcpHost              = mustGetenv("DB_TCP_HOST")
	//dbPort                 = mustGetenv("DB_PORT")
	dbName := mustGetenv("DB_NAME")
	instanceConnectionName := mustGetenv("INSTANCE_CONNECTION_NAME")

	db, err := psdb.NewSocket(dbUser, dbPwd, instanceConnectionName, dbName)
	if err != nil {
		log.Fatalf("Error connecting to Database %s", err)
	}

	date, err := getDate(db)
	if err != nil {
		log.Fatalf("Error getting last stock update date from database %s", err)
	}

	text, err := util.Fetch(url)
	if err != nil {
		log.Fatalf("error[%v]", err)
	}
	_, items := scraper.Scrape(text)

	sort.Slice(items, func(i, j int) bool {
		return items[i].Date.Before(items[j].Date)
	})

	//Filter old inventory updates
	items = scraper.Filter(items, func(row scraper.Status) bool {
		return row.Date.After(*date)
	})

	//get instock
	instock := scraper.Filter(items, func(row scraper.Status) bool {
		return strings.Contains(row.Data, "In Stock")
	})

	//get out of stock
	outOfstock := scraper.Filter(items, func(row scraper.Status) bool {
		return strings.Contains(row.Data, "Out of Stock")
	})

	if err := sendMessage("PS5 InStock Update on "+date.Format(util.DateTempate), instock); err != nil {
		log.Printf("unable to send slack instock of stock update")
	}
	if err := sendMessage("PS5 Out of Stock Update on "+date.Format(util.DateTempate), outOfstock); err != nil {
		log.Printf("unable to send slack out of stock update")
	}

	if err = psdb.SaveCronResults(db, items); err != nil {
		log.Printf("error saving cron resutls %s", err)
	}
	if err = psdb.SaveAvailableResults(db, instock); err != nil {
		log.Printf("error saving cron resutls %s", err)
	}

	lastdate := items[len(items)-1].Date.Format(util.DateTempate)
	log.Printf("Updating config to date: %v", lastdate)
	if err := psdb.SaveConfig(db, util.CacheKey, lastdate); err != nil {
		log.Fatalf(" unable to save last date to db: %v", err)
	}
}

func sendMessage(header string, s []scraper.Status) error {

	if len(s) > 0 {
		mardownBlocks := make([]slack.Block, 0)
		mardownBlocks = append(mardownBlocks, sc.Header(header))
		mardownBlocks = append(mardownBlocks, sc.Divider())
		for _, row := range sc.SectionTextBlocks(s) {
			mardownBlocks = append(mardownBlocks, row)
		}
		list := sc.Message(mardownBlocks)
		_, _, err := api.PostMessage(slack_channel, slack.MsgOptionText("", false), list)

		return err
	}

	return nil
}

// mustGetEnv is a helper function for getting environment variables.
// Displays a warning if the environment variable is not set.
func mustGetenv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("Warning: %s environment variable not set.\n", k)
	}
	return v
}

func main() {

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/event/", eventHandler)
	http.HandleFunc("/crawl/", crawl)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
