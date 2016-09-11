package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/cgi"
	timepkg "time"
	"zng.jp/tv"
)

func processGetJson(writer http.ResponseWriter, request *http.Request) {
	data, err := readData()
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(writer).Encode(data); err != nil {
		log.Print("Encode failed: %v", err)
		return
	}
}

func processGetEventStream(writer http.ResponseWriter, request *http.Request) {
	listenCancel := make(chan struct{})
	defer close(listenCancel)

	acceptDone, err := listenData(listenCancel)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")

	var closeDone <-chan bool
	if closeNotifier, ok := writer.(http.CloseNotifier); ok {
		closeDone = closeNotifier.CloseNotify()
	}

	for {
		timer := timepkg.NewTimer(timepkg.Second * 10)
		defer timer.Stop()

		select {
		case _, ok := <-acceptDone:
			if !ok {
				return
			}
			if _, err := io.WriteString(writer, "data: ok\n\n"); err != nil {
				log.Print("WriteString failed: %v", err)
				return
			}
		case <-timer.C:
			if _, err := io.WriteString(writer, "\n"); err != nil {
				log.Print("WriteString failed: %v", err)
				return
			}
		case <-closeDone:
			return
		}

		if flusher, ok := writer.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

func processGetHtml(writer http.ResponseWriter, request *http.Request) {
	data, err := readData()
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := renderIndex(data, writer); err != nil {
		log.Print("Render failed: %v", err)
		return
	}
}

func processGet(writer http.ResponseWriter, request *http.Request) {
	url := request.URL
	query := url.Query()
	mode := query.Get("mode")
	switch mode {
	case "json":
		processGetJson(writer, request)
	case "event-stream":
		processGetEventStream(writer, request)
	case "html":
		processGetHtml(writer, request)
	default:
		http.Error(writer, "Unknown mode: "+mode, http.StatusBadRequest)
	}
}

func processPostJson(writer http.ResponseWriter, request *http.Request) {
	if request.Body == nil {
		http.Error(writer, "Request body is nil", http.StatusInternalServerError)
		return
	}

	newData := &tv.Data{}
	if err := json.NewDecoder(request.Body).Decode(newData); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := writeData(newData); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func processPostHtml(writer http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	newData, err := parseRuleConfig(request.PostForm)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := writeData(newData); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(writer, request, "./?mode=html", http.StatusSeeOther)
}

func processPost(writer http.ResponseWriter, request *http.Request) {
	url := request.URL
	query := url.Query()
	mode := query.Get("mode")
	switch mode {
	case "json":
		processPostJson(writer, request)
	case "html":
		processPostHtml(writer, request)
	default:
		http.Error(writer, "Unknown mode: "+mode, http.StatusBadRequest)
	}
}

type handler struct {
}

func (*handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method == "GET" || request.Method == "HEAD" {
		processGet(writer, request)
	} else if request.Method == "POST" {
		processPost(writer, request)
	} else {
		http.Error(writer, "Method "+request.Method+" not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	handler := &handler{}
	if err := cgi.Serve(handler); err != nil {
		log.Fatal(err)
	}
}
