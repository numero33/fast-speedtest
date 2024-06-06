package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func getTargets() ([]Target, error) {
	token, err := getToken()
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(fmt.Sprintf("https://api.fast.com/netflix/speedtest/v2?https=true&token=%s&urlCount=5", token))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	var response APIResponse

	if err := decoder.Decode(&response); err != nil {
		return nil, err
	}

	return response.Targets, nil
}

func getToken() (string, error) {
	// find script
	resp, err := http.Get("https://fast.com")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	scriptFound := reFastComScript.FindStringSubmatch(string(b))
	//fmt.Println(scriptFound[1])

	// load script
	resp, err = http.Get("https://fast.com/" + scriptFound[1])
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// extract token
	tokenFound := reFastComToken.FindStringSubmatch(string(b))
	//fmt.Println(tokenFound[1])

	return tokenFound[1], nil
}
