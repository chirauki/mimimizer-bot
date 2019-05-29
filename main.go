package main

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v2"
)

func mimimize(in string) string {
	result := in

	r, _ := regexp.Compile("[aàáeèéoòóuùú]")
	rCaps, _ := regexp.Compile("[AÀÁEÈÉIÍÌOÒÓUÙÚ]")

	result = r.ReplaceAllString(result, "i")
	result = rCaps.ReplaceAllString(result, "I")

	return result
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Warning("Error loading .env file. Will not be used.")
	}

	token, ok := os.LookupEnv("TB_KEY")
	if !ok {
		log.Fatal("Env var with telegram token key is not found!")
		return
	}

	var b *tb.Bot

	port, portExists := os.LookupEnv("PORT")
	if !portExists {
		log.Info("PORT env var does not exist. Using long poller")

		b, err = tb.NewBot(tb.Settings{
			Token:  token,
			Poller: &tb.LongPoller{Timeout: 10 * time.Second},
		})

		if err != nil {
			log.Fatal(err)
			return
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

		b, err = tb.NewBot(pref)
		if err != nil {
			log.Fatalln(err)
		}

	}

	b.Handle(tb.OnQuery, func(q *tb.Query) {
		log.Info(fmt.Sprintf("Received inline query from user '%s' with text: %s", q.From.Username, q.Text))

		result := &tb.ArticleResult{
			Text:  mimimize(q.Text),
			Title: mimimize(q.Text),
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
		log.Info(fmt.Sprintf("Received message in chat ID %d with message ID %d", m.Chat.ID, m.ID))

		if m.IsReply() {
			originalMessage := m.ReplyTo
			mimimized := mimimize(originalMessage.Text)

			err = b.Delete(m)
			if err != nil {
				log.Error(err)
			}
			b.Reply(originalMessage, mimimized)
		}
	})

	b.Start()
}
