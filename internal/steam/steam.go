package steam

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"fmt"
	"net/url"
	"strings"
)

var invalidCustomURL = errors.New("steam: invalid custom URL")
var invalidCustomURLPath = errors.New("steam: custom URL has unexpected path")

type IDByCustomURLResponse struct {
	Response struct {
		SteamID string `json:"steamid"`
		Success int    `json:"success"`
	} `json:"response"`
}

// Steam represents the Steam WebAPI
type Steam struct {
	APIKey string
}

func New(apiKey string) *Steam {
	return &Steam{APIKey: apiKey}
}

func (s *Steam) GetSteamIDByCustomURL(customURL string) (string, error) {
	customURLParsed, err := url.Parse(customURL)

	if err != nil {
		return "", fmt.Errorf("invalid custom URL: %w", invalidCustomURL)
	}

	pathParts := strings.Split(strings.TrimRight(customURLParsed.Path, "/"), "/")

	if len(pathParts) < 1 {
		return "", invalidCustomURLPath
	}

	queryParams := url.Values{
		"key":       []string{s.APIKey},
		"vanityurl": []string{pathParts[len(pathParts)-1]},
	}
	apiURL := "https://api.steampowered.com/ISteamUser/ResolveVanityURL/v0001/?" + queryParams.Encode()
	resp, err := http.Get(apiURL)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var response IDByCustomURLResponse

	_ = json.Unmarshal(body, &response)

	return response.Response.SteamID, nil
}
