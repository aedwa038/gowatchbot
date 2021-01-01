package scraper

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aedwa038/ps5watcherbot/util"
)

type Heading []string

type Status struct {
	Date time.Time `json:"time"`
	Data string    `json:"data"`
}

func Scrape(text string) (Heading, []Status) {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))
	if err != nil {
		log.Fatal(err)
	}

	var row []string
	var headings Heading
	var rows [][]string
	var s []Status

	doc.Find("table").Each(func(index int, tablehtml *goquery.Selection) {
		tablehtml.Find("tr").Each(func(indextr int, rowhtml *goquery.Selection) {
			rowhtml.Find("th").Each(func(indexth int, tableheading *goquery.Selection) {
				headings = append(headings, tableheading.Text())
			})
			rowhtml.Find("td").Each(func(indexth int, tablecell *goquery.Selection) {
				row = append(row, tablecell.Text())
			})
			rows = append(rows, row)
			row = nil
		})
	})

	fmt.Println("####### headings = ", len(headings), headings)
	fmt.Println("####### rows = ", len(rows))

	for i := 0; i < len(rows); i++ {
		if len(rows[i]) >= 2 {
			t, err := time.Parse(util.DateTempate, rows[i][0])
			if err != nil {
				log.Fatalf("error %s", err)
			}
			s = append(s, Status{Date: t, Data: rows[i][1]})
		}
	}
	return headings, s
}

func Filter(vs []Status, f func(Status) bool) []Status {
	vsf := make([]Status, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}
