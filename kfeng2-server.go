package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

func status(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Map that will later be turned into JSON object
	resp := make(map[string]string)

	// Check request endpoint, only respond to "/kfeng2/status", others will get 404
	if req.URL.String() != "/kfeng2/status" {
		resp["HTTP status"] = "404 Not Found"
		w.WriteHeader(http.StatusNotFound)
	} else if req.Method != "GET" {
		resp["HTTP status"] = "405 Method Not Allowed"
		w.WriteHeader(http.StatusMethodNotAllowed)
	} else {
		resp["Local time"] = time.Now().String()
		resp["HTTP status"] = "200 OK"
		w.WriteHeader(http.StatusOK)
	}

	// Marshal into json object
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}

	// Write the response
	w.Write(jsonResp)

	// Stuff that should be sent to Loggly
	fmt.Println("\nRespond status: ", resp["HTTP status"])
	fmt.Println("Request method: ", req.Method)     // This gives the calling method
	fmt.Println("Request url: ", req.URL)           // This gives the request path
	fmt.Println("Request source: ", req.RemoteAddr) // This gives the caller ip address

	var logglyURL = os.Getenv("Loggly_Token")
	var data = url.Values{
		"method": {req.Method},
		"source": {req.RemoteAddr},
		"path":   {req.URL.String()},
		"status": {resp["HTTP status"]},
	}

	_, err = http.PostForm(logglyURL, data)
	if err != nil {
		panic(err)
	}
}

func main() {
	http.HandleFunc("/", status)
	http.ListenAndServe(":8090", nil)
}
