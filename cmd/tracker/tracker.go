package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var msInDay uint = 86400000

func main() {
	godotenv.Load()

	token := os.Getenv("CLICKUP_TOKEN")
	if token == "" {
		log.Fatal("No Clickup access token provided. Please set the .env file")
	}

	team := os.Getenv("CLICKUP_TEAM")
	if team == "" {
		log.Fatal("No Clickup team id provided. Please set the .env file")
	}

	entries := fetchEntries(token, team)
	currentTask := hasOngoingTask(token, team)

	if currentTask != nil {
		entries = append(entries, *currentTask)
	}

	total := sumEntries(entries)
	hours := math.Floor(total.Hours())
	minutes := math.Floor(total.Minutes() - (hours)*60)

	overtimeFlag := ""
	if hours >= 8 {
		overtimeFlag = " [!]"
	}

	ongoingFlag := ""
	if currentTask != nil {
		ongoingFlag = " [+]"
	}

	fmt.Printf("%v:%v%s%s\n",
		hours,
		minutes,
		ongoingFlag,
		overtimeFlag,
	)
}

func fetchEntries(token string, team string) []Entry {
	clickupApiUrl := "https://api.clickup.com/api/v2"

	now := time.Now()
	start := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0, 0, 0, 0,
		now.Location(),
	).Unix() * 1000
	end := start + int64(msInDay)

	reqUrl := fmt.Sprintf(
		"%v/team/%v/time_entries?is_billable=false&start_date=%v&end_date=%v",
		clickupApiUrl, team, start, end,
	)

	client := &http.Client{}

	req, err := http.NewRequest(
		"GET",
		reqUrl,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to build a request to clickup's API")
	}

	req.Header.Add("Authorization", token)

	res, err := client.Do(req)
	if err != nil {
		log.Fatal("Failed to request clickup's API")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Failed to read response's body")
	}

	data := ClickUpEntriesResponse{}

	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Printf("%v\n", err.Error())
		log.Fatal("Failed to parse response body")
	}

	return data.Data
}

func sumEntries(entries []Entry) time.Duration {
	durMs := 0

	for _, entry := range entries {
		start, _ := strconv.Atoi(entry.Start)
		end := int(time.Now().Unix() * 1000)
		if entry.End != nil {
			end, _ = strconv.Atoi(*entry.End)
		}

		durMs += int(math.Abs(float64(end)) - math.Abs(float64(start)))
	}

	return time.Duration(durMs) * time.Millisecond
}

func hasOngoingTask(token string, team string) *Entry {
	clickupApiUrl := "https://api.clickup.com/api/v2"

	reqUrl := fmt.Sprintf(
		"%s/team/%v/time_entries/current",
		clickupApiUrl, team,
	)

	client := &http.Client{}

	req, err := http.NewRequest(
		"GET",
		reqUrl,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to build a request to clickup's API")
	}

	req.Header.Add("Authorization", token)

	res, err := client.Do(req)
	if err != nil {
		log.Fatal("Failed to request clickup's API")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Failed to read response's body")
	}

	data := ClickUpCurrentTaskResponse{}

	err = json.Unmarshal(body, &data)
	if err != nil {
		slog.Error(fmt.Sprintf("%v\n", err.Error()))
		log.Fatal("Failed to parse response body")
	}

	return data.Data
}

type Entry struct {
	Start string  `json:"start"`
	End   *string `json:"end"`
}

type ClickUpEntriesResponse struct {
	Data []Entry `json:"data"`
}

type ClickUpCurrentTaskResponse struct {
	Data *Entry `json:"data"`
}
