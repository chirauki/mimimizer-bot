package main

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v2"
	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
)

func mimimize(in string) string {
	r, _ := regexp.Compile("[aàáeèêoòóuùú]")
	out := r.ReplaceAllString(in, "i")
	return out
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Warning("Error loading .env file. Will not be used.")
	}

	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl.html", nil)
	})

	router.Run(":" + port)

	token, ok := os.LookupEnv("TB_KEY")
	if !ok {
		log.Fatal("Env var with telegram token key is not found!")
		return
	}

	b, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle(tb.OnQuery, func(q *tb.Query) {
		log.Info(fmt.Sprintf("%v", q))

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
			b.Reply(originalMessage, mimimized)
		}
	})

	b.Start()
}
