package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var dg *discordgo.Session

type YahooResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
			} `json:"meta"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	dtoken := os.Getenv("dtoken")
	if dtoken == "" {
		panic("no discord token")
	}

	var err error
	dg, err = discordgo.New("Bot " + dtoken)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Discord session")
		return
	}

	err = dg.Open()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open connection")
		return
	}
	log.Info().Msg("Bot is running")

	go loop()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	log.Info().Msg("Shutting down")
	dg.Close()
}

func loop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	runUpdates()

	for {
		select {
		case <-ticker.C:
			runUpdates()
		}
	}
}

func runUpdates() {
	update("SP&500", "GSPC", "1329033384378503198")
	update("DJI", "DJI", "1329036261553340426")
	update("VIX", "VIX", "1329036335037681715")
	update("5Year", "FVX", "1329036378637340673")
	update("FTSE100", "FTSC", "1329036854783381534")
	update("NASDAQ", "IXIC", "1329036890275577856")
}

func update(name string, ticker string, channel string) {
	url := "https://query2.finance.yahoo.com/v8/finance/chart/%5E" + ticker

	resp, err := http.Get(url)
	if err != nil {
		log.Error().Err(err).
			Str("name", name).
			Str("ticker", ticker).
			Str("channel", channel).
			Msg("Error making request")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).
			Str("name", name).
			Str("ticker", ticker).
			Str("channel", channel).
			Msg("Error reading response")
		return
	}

	var yahooResp YahooResponse
	err = json.Unmarshal(body, &yahooResp)
	if err != nil {
		log.Error().Err(err).
			Str("name", name).
			Str("ticker", ticker).
			Str("channel", channel).
			Msg("Error parsing JSON")
		return
	}

	if len(yahooResp.Chart.Result) == 0 {
		log.Error().Err(err).
			Str("name", name).
			Str("ticker", ticker).
			Str("channel", channel).
			Msg("No data found in response")
		return
	}

	price := yahooResp.Chart.Result[0].Meta.RegularMarketPrice
	_, err = dg.ChannelEdit(channel, &discordgo.ChannelEdit{
		Name: name + ": " + fmt.Sprintf("%.2f", price),
	})
	if err != nil {
		log.Error().Err(err).
			Str("name", name).
			Str("ticker", ticker).
			Str("channel", channel).
			Msg("Error setting channel name")
	}
}
