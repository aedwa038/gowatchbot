package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"

	psdb "github.com/aedwa038/ps5watcherbot/mysql"
	"github.com/aedwa038/ps5watcherbot/scraper"
	sc "github.com/aedwa038/ps5watcherbot/slack"
	"github.com/aedwa038/ps5watcherbot/util"
	_ "github.com/go-sql-driver/mysql"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

var indexTmpl = template.Must(template.ParseFiles("index.html"))
var signingSecret = mustGetenv("SLACK_SIGNING_KEY")
var api = slack.New(mustGetenv("SLACK_TOKEN"))
var url = mustGetenv("URL")

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
	} else if ch == "" {
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
	slack_channel := mustGetenv("DEFAULT_SLACK_CHANNEL")
	//dbTcpHost              = mustGetenv("DB_TCP_HOST")
	//dbPort                 = mustGetenv("DB_PORT")
	dbName := mustGetenv("DB_NAME")
	instanceConnectionName := mustGetenv("INSTANCE_CONNECTION_NAME")

	db, err := psdb.NewSocket(dbUser, dbPwd, instanceConnectionName, dbName)
	if err != nil {
		fmt.Fprintf(w, "Error %s", err)
		return
	}

	date, err := getDate(db)
	if err != nil {
		fmt.Fprintf(w, "Error %s", err)
		return
	}

	text, err := util.Fetch(url)
	if err != nil {
		log.Fatalf("error[%v]", err)
	}
	_, items := scraper.Scrape(text)

	instock := scraper.Filter(items, func(row scraper.Status) bool {
		return strings.Contains(row.Data, "In Stock")
	})

	instock = scraper.Filter(instock, func(row scraper.Status) bool {
		return row.Date.After(*date)
	})

	sort.Slice(instock, func(i, j int) bool {
		return instock[i].Date.Before(instock[j].Date)
	})

	if len(instock) > 0 {
		mardownBlocks := make([]slack.Block, 0)
		mardownBlocks = append(mardownBlocks, sc.Header("PS5 InStock Update on "+date.Format(util.DateTempate)))
		mardownBlocks = append(mardownBlocks, sc.Divider())
		for _, row := range sc.SectionTextBlocks(instock) {
			mardownBlocks = append(mardownBlocks, row)
		}
		list := sc.Message(mardownBlocks)
		api.PostMessage(slack_channel, slack.MsgOptionText("", false), list)

	} else {
		//api.PostMessage("ps5test", slack.MsgOptionText("No new PS5 stock updates Found :(", false))
		log.Print("No new PS5 stock updates")
	}

	if err = psdb.SaveCronResults(db, items); err != nil {
		fmt.Fprintf(w, "Error %s", err)
	}
	if err = psdb.SaveAvailableResults(db, instock); err != nil {
		fmt.Fprintf(w, "Error %s", err)
	}
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
