package db

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"zng.jp/tv"
)

func FetchData() (*tv.Data, error) {
	response, err := http.Get("http://zng.jp/tv/tvctl.cgi?mode=json")
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return nil, errors.New("Server returned on-OK status: " + strconv.Itoa(response.StatusCode) + " " + string(body))
	}

	data := &tv.Data{}
	if err := json.NewDecoder(response.Body).Decode(data); err != nil {
		return nil, err
	}

	return data, nil
}

func ListenData(notificationQueue chan<- struct{}) error {
	response, err := http.Get("http://zng.jp/tv/tvctl.cgi?mode=event-stream")
	if err != nil {
		return err
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return errors.New("Server returned on-OK status: " + strconv.Itoa(response.StatusCode) + " " + string(body))
	}

	scanner := bufio.NewScanner(response.Body)
	data := ""
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			data = strings.TrimSuffix(data, "\n")
			if data != "" {
				notificationQueue <- struct{}{}
				data = ""
			}
		} else {
			fieldValue := strings.SplitN(line, ":", 2)
			field := fieldValue[0]
			value := ""
			if len(fieldValue) == 2 {
				value = fieldValue[1]
			}
			value = strings.TrimPrefix(value, " ")

			if field == "data" {
				data += value
			}
		}
	}

	return scanner.Err()
}

func PostData(cancel <-chan struct{}, data *tv.Data) error {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(data); err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, "http://zng.jp/tv/tvctl.cgi?mode=json", buf)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Length", strconv.Itoa(buf.Len()))
	request.Cancel = cancel

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return errors.New("Server returned on-OK status: " + strconv.Itoa(response.StatusCode) + " " + string(body))
	}

	return nil
}
