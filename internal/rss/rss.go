package rss

import (
	"context"
	"encoding/xml"
	"html"
	"io"
	"net/http"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func FetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "gator")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	feed := RSSFeed{}

	err = xml.Unmarshal(resBody, &feed)
	if err != nil {
		return nil, err
	}

	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	for _, item := range feed.Channel.Item {
		item.Title = html.UnescapeString(item.Title)
		item.Description = html.UnescapeString(item.Description)
	}

	return &feed, nil
}
// type Config struct {
// 	DbUrl string `json:"db_url"`
// 	CurrentUserName string `json:"current_user_name"`
// }
//
// func Read() (*Config, error) {
// 	filePath, err := getConfigFilePath()
// 	if err != nil {
// 		return nil, fmt.Errorf("error while reading config")
// 	}
//
// 	file, err := os.Open(filePath)
// 	if err != nil {
// 		return nil, fmt.Errorf("error while reading config")
// 	}
// 	defer file.Close()
//
// 	var payload Config
// 	decoder := json.NewDecoder(file)
// 	if err := decoder.Decode(&payload); err != nil {
// 		return nil, fmt.Errorf("error while reading config")
// 	}
//
// 	return &payload, nil
// }
//
// func (c Config) SetUser(userName string) error {
// 	filePath, err := getConfigFilePath()
// 	if err != nil {
// 		return fmt.Errorf("error while setting user")
// 	}
//
// 	file, err := os.Create(filePath)
// 	if err != nil {
// 		return fmt.Errorf("error while setting user")
// 	}
// 	defer file.Close()
//
// 	c.CurrentUserName = userName
//
// 	encoder := json.NewEncoder(file)
// 	if err := encoder.Encode(c); err != nil {
// 		return fmt.Errorf("error while setting user")
// 	}
//
// 	return nil
// }
//
// func getConfigFilePath() (string, error) {
// 	home, err := os.UserHomeDir()
//
// 	if err != nil {
// 		return "", fmt.Errorf("error while reading config path")
// 	}
//
// 	return home + "/.gatorconfig.json", nil
// }
