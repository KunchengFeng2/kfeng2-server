package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/gorilla/mux"
)

var loggly_Token string
var database *dynamodb.DynamoDB
var tableName string

func init() {
	// Load environment variables
	// godotenv.Load("csc482.env")
	loggly_Token = os.Getenv("Loggly_Token")
	// fmt.Println("Loggly_Token: ", loggly_Token)

	// Establish a new connection, the credentials should be set as environment variables
	sess, err := session.NewSession()
	if err != nil {
		fmt.Println("Problem occured when forming new session!")
		os.Exit(1)
	}
	database = dynamodb.New(sess)

	tableName = "Kfeng2_MC_Servers"
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/kfeng2/all", all).Methods("GET")
	router.HandleFunc("/kfeng2/status", status).Methods("GET")
	router.HandleFunc("/kfeng2/search", search).Methods("GET")
	router.NotFoundHandler = http.HandlerFunc(notFound)
	http.ListenAndServe(":8080", router)
}

func all(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	// Scan everything on the table
	result, err := database.Scan(&dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		log.Fatalf("Query API call failed: %s", err)
	}

	// Put database scan result into golang strucutre
	var response_golang []ServerStatus
	for _, i := range result.Items {
		item_golang := ServerStatus{}

		err = dynamodbattribute.UnmarshalMap(i, &item_golang)
		if err != nil {
			log.Fatalf("Got error unmarshalling: %s", err)
		}

		response_golang = append(response_golang, item_golang)
	}

	// Marshal golang structure into json structure
	var response_json, _ = json.Marshal(response_golang)

	// Respond back into the user
	writer.Write(response_json)

	// Send to loggly
	sendToLoggly("200 OK", request)
}

func status(writer http.ResponseWriter, request *http.Request) {
	// Write to header
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	// Scanning is too expensive, using DescribeTable as suggested by Professor Early.
	description, err := database.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		log.Fatalf("Failed to communicate with database: %s", err)
	}

	// fmt.Println("Table description:\n ", description)
	var number = *description.Table.ItemCount

	// Prepair response
	response := make(map[string]string)
	response["recordCount"] = strconv.FormatInt(number, 10)
	response["table"] = tableName

	// Marshal response into json object
	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}

	// Write the response to the body
	writer.Write(jsonResp)

	// send this to loggly
	sendToLoggly("200 OK", request)
}

func search(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")

	var message string

	var query = request.URL.Query()
	var filters, present = query["Hostname"]
	if !present || len(filters) == 0 || len(filters) > 1 {
		message = "400 Bad Request"
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(message))
	} else {
		// The request is valid, but result is not gaurenteed
		writer.WriteHeader(http.StatusOK)
		message = "200 OK"

		// Filter the request url
		var text = filters[0]
		text = strings.ReplaceAll(text, " ", "")
		text = strings.ReplaceAll(text, ";", "")
		text = strings.ReplaceAll(text, "=", "")

		// Now ready for dynamoDB
		var queryInput = &dynamodb.QueryInput{
			TableName: aws.String(tableName),
			KeyConditions: map[string]*dynamodb.Condition{
				"Hostname": {
					ComparisonOperator: aws.String("EQ"),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							S: aws.String(text),
						},
					},
				},
			},
		}
		var result, err = database.Query(queryInput)
		if err != nil {
			log.Fatalf("Got error calling GetItem: %s", err)
		}

		// Put database query result into golang strucutre
		var response_golang []ServerStatus
		for _, i := range result.Items {
			item_golang := ServerStatus{}

			err = dynamodbattribute.UnmarshalMap(i, &item_golang)
			if err != nil {
				log.Fatalf("Got error unmarshalling: %s", err)
			}

			response_golang = append(response_golang, item_golang)
		}

		// golang structure into json structure
		var response_json, _ = json.Marshal(response_golang)

		// Report the result back to user
		writer.Write(response_json)

	}

	sendToLoggly(message, request)
}

func notFound(writer http.ResponseWriter, request *http.Request) {
	// This method handles the default traffic, includes the wrong methods

	var message string
	if request.Method == "GET" {
		message = "404 Not Found"
		writer.WriteHeader(http.StatusNotFound)
	} else {
		message = "405 Method Not Allowed"
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}

	writer.Write([]byte(message))

	// Report this event
	sendToLoggly(message, request)
}

func sendToLoggly(message string, request *http.Request) {
	// Stuff that should be sent to Loggly
	// fmt.Println("Respond message: ", message)
	// fmt.Println("Request method: ", request.Method)     // This gives the calling method
	// fmt.Println("Request url: ", request.URL)           // This gives the request path
	// fmt.Println("Request source: ", request.RemoteAddr) // This gives the caller ip address
	// fmt.Println()

	var data = url.Values{
		"method":  {request.Method},
		"source":  {request.RemoteAddr},
		"path":    {request.URL.String()},
		"message": {message},
	}

	response, err := http.PostForm(loggly_Token, data)
	if err != nil {
		panic(err)
	}

	// Memory leak if not ?
	defer response.Body.Close()
}

type ServerStatus struct {
	IP       string
	Version  string
	Online   bool
	Hostname string
	Players  struct {
		Online int
		Max    int
	}
	Time string
}
