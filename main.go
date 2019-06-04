package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v2"
)

type Mimimizer struct {
	bot        *tb.Bot
	stats      map[int]int
	rudes      []string
	angryLimit int
}

func NewMimimizer() (Mimimizer, error) {
	m := Mimimizer{}

	m.stats = make(map[int]int)

	// DEV load env from file
	err := godotenv.Load()
	if err != nil {
		log.Warning("Error loading .env file. Will not be used.")
	}

	// Without a Telegram API key, anything beyond this point makes
	// no sense...
	token, ok := os.LookupEnv("TB_KEY")
	if !ok {
		return m, errors.New("env var with telegram token key is not found")
	}

	// Assuming PORT variable is set by Heroku.
	// If it exists, we'll use a webhook.
	// If it doesn't (local) use long polling.
	port, portExists := os.LookupEnv("PORT")
	if !portExists {
		log.Info("PORT env var does not exist. Using long poller")

		m.bot, err = tb.NewBot(tb.Settings{
			Token:  token,
			Poller: &tb.LongPoller{Timeout: 10 * time.Second},
		})

		if err != nil {
			return m, err
		}
	} else {
		log.Info("PORT env var exists. Using webhook poller")
		publicURL := os.Getenv("PUBLIC_URL")
		log.Info(fmt.Sprintf("Webhook public URL is: %s", publicURL))

		webhook := &tb.Webhook{
			Listen:   ":" + port,
			Endpoint: &tb.WebhookEndpoint{PublicURL: publicURL},
		}

		pref := tb.Settings{
			Token:  token,
			Poller: webhook,
		}

		m.bot, err = tb.NewBot(pref)
		if err != nil {
			return m, err
		}
	}

	// Get the angry count.
	// Each time a user asks for mimimi ANGRY_COUNT
	// times, he/she will get an rude response instead.
	countEnv, ok := os.LookupEnv("ANGRY_COUNT")
	if !ok {
		m.angryLimit = 10
	} else {
		m.angryLimit, err = strconv.Atoi(countEnv)
		if err != nil {
			log.Warn(fmt.Sprintf("%s is not a number so I will use 10... just because.", countEnv))
			m.angryLimit = 10
		}
	}

	// Get the list of rude responses from env vars
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		if strings.HasPrefix(pair[0], "RUDE") {
			m.rudes = append(m.rudes, pair[1])
		}
	}
	if len(m.rudes) == 0 {
		m.rudes = append(m.rudes, "... por que no te callas?...")
	}

	return m, nil
}

func (m *Mimimizer) beRude() string {
	rand.Seed(time.Now().Unix())
	return m.rudes[rand.Intn(len(m.rudes))]
}

func (m *Mimimizer) mimimize(in string) string {
	result := in

	r, _ := regexp.Compile("[aàáeèéoòóuùú]")
	rCaps, _ := regexp.Compile("[AÀÁEÈÉIÍÌOÒÓUÙÚ]")

	result = r.ReplaceAllString(result, "i")
	result = rCaps.ReplaceAllString(result, "I")

	return result
}

func main() {
	mimimizer, err := NewMimimizer()
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	b := mimimizer.bot

	b.Handle(tb.OnQuery, func(q *tb.Query) {
		log.Info(fmt.Sprintf("Received inline query from user '%s' with text: %s", q.From.Username, q.Text))

		result := &tb.ArticleResult{
			Text:  mimimizer.mimimize(q.Text),
			Title: mimimizer.mimimize(q.Text),
		}
		results := make(tb.Results, 1)
		results[0] = result

		err := b.Answer(q, &tb.QueryResponse{
			Results:   results,
			CacheTime: 60, // a minute
		})

		if err != nil {
			fmt.Println(err)
		}
	})

	b.Handle("/mimimi", func(m *tb.Message) {
		log.Infof("Received message in chat ID %s with message ID %d", m.Chat.Title, m.ID)

		if m.IsReply() {
			count, ok := mimimizer.stats[m.Sender.ID]
			log.Infof("User ID %d is actually %s", m.Sender.ID, m.Sender.Username)
			log.Infof("MAP: %v", mimimizer.stats)

			if !ok || count%mimimizer.angryLimit != 0 {
				log.Infof("User %s is at %d", m.Sender.Username, count)
				originalMessage := m.ReplyTo
				mimimized := mimimizer.mimimize(originalMessage.Text)

				err = b.Delete(m)
				if err != nil {
					log.Error(err)
				}
				b.Reply(originalMessage, mimimized)

				if !ok {
					log.Info("Never used it before, set to 1")
					mimimizer.stats[m.Sender.ID] = 1
				} else {
					log.Info("Incrementing count")
					mimimizer.stats[m.Sender.ID] = count + 1
				}
			} else {
				log.Infof("User %s is at %d and angry count is %d", m.Sender.Username, count, mimimizer.angryLimit)
				rude := mimimizer.beRude()
				log.Info("ZASCA! ", rude)
				b.Reply(m, fmt.Sprintf("@%s , %s", m.Sender.Username, rude))
				log.Info("Incrementing count")
				mimimizer.stats[m.Sender.ID] = count + 1
			}
		}
	})

	b.Start()
}
