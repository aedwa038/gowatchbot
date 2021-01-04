//package slack is mostly a wrapper to the slack-go package
package slack

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aedwa038/gowatcherbot/scraper"
	"github.com/aedwa038/gowatcherbot/util"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

//NewSlackClient creates a new slack client and returns it
func NewSlackClient(token string) *slack.Client {
	api := slack.New(token)
	return api
}

//GetRequestBody retreives the slack request from the http request
func GetRequestBody(r *http.Request) ([]byte, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return []byte{}, nil
	}
	return body, nil
}

// ValidateRequestBody validates the slack request
func ValidateRequestBody(r http.Header, body []byte, signingSecret string) error {
	sv, err := slack.NewSecretsVerifier(r, signingSecret)
	if err != nil {
		return err
	}
	if _, err := sv.Write(body); err != nil {
		return err
	}
	if err := sv.Ensure(); err != nil {
		return err
	}

	return nil
}

//GetEvents gets the slack event inside of the request body
func GetEvents(body []byte) (slackevents.EventsAPIEvent, error) {
	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		return slackevents.EventsAPIEvent{}, err
	}
	return eventsAPIEvent, nil
}

//HandleVerifcationRequest helper function for handling slack verifications
func HandleVerifcationRequest(e slackevents.EventsAPIEvent, body []byte) (string, error) {
	if e.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		if err := json.Unmarshal([]byte(body), &r); err != nil {
			return "", err
		}
		return r.Challenge, nil
	}
	return "", nil
}

func Header(header string) slack.Block {
	headerText := slack.NewTextBlockObject("plain_text", header, false, false)
	headerSection := slack.NewSectionBlock(headerText, nil, nil)
	return headerSection
}

func Divider() slack.Block {
	return slack.NewDividerBlock()
}

func SectionTextBlock(i scraper.Status) slack.Block {
	msg := fmt.Sprintf("%v: <%v|%v>\n", i.Date, util.GetURL(i.Data), i.Data)
	t := slack.NewTextBlockObject("mrkdwn", msg, false, false)
	h := slack.NewSectionBlock(t, nil, nil)
	return h
}

func SectionTextBlocks(instock []scraper.Status) []slack.Block {

	mardownBlocks := make([]slack.Block, 0)
	for _, row := range instock {
		h := SectionTextBlock(row)
		mardownBlocks = append(mardownBlocks, h)
	}

	return mardownBlocks
}

func Message(blocks []slack.Block) slack.MsgOption {
	list := slack.MsgOptionBlocks(
		blocks...,
	)
	return list
}
