/*
Copyright 2021 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/dapr/dapr/tests/apps/utils"
)

const (
	appPort = 3000
	// pubsubRawSubTopic     = "pubsub-raw-sub-topic-http"
	// pubsubCESubTopic      = "pubsub-ce-sub-topic-http"
	pubsubRawBulkSubTopic = "pubsub-raw-bulk-sub-topic-http"
	pubsubCEBulkSubTopic  = "pubsub-ce-bulk-sub-topic-http"
	PubSubEnvVar          = "DAPR_TEST_PUBSUB_NAME"
)

var pubsubkafkaName = "kafka-pubsub-comp"

func init() {
	if psName := os.Getenv(PubSubEnvVar); len(psName) != 0 {
		pubsubkafkaName = psName
	}
}

type appResponse struct {
	// Status field for proper handling of errors form pubsub
	Status    string `json:"status,omitempty"`
	Message   string `json:"message,omitempty"`
	StartTime int    `json:"start_time,omitempty"`
	EndTime   int    `json:"end_time,omitempty"`
}

type receivedMessagesResponse struct {
	// ReceivedByTopicRawSub     []string `json:"pubsub-raw-sub-topic"`
	// ReceivedByTopicCESub      []string `json:"pubsub-ce-sub-topic"`
	ReceivedByTopicRawBulkSub []string `json:"pubsub-raw-bulk-sub-topic"`
	ReceivedByTopicCEBulkSub  []string `json:"pubsub-ce-bulk-sub-topic"`
}

type subscription struct {
	PubsubName      string            `json:"pubsubname"`
	Topic           string            `json:"topic"`
	Route           string            `json:"route"`
	DeadLetterTopic string            `json:"deadLetterTopic"`
	Metadata        map[string]string `json:"metadata"`
}

type BulkRawMessage struct {
	Entries  []BulkMessageRawEntry `json:"entries"`
	Topic    string                `json:"topic"`
	Metadata map[string]string     `json:"metadata"`
}

type BulkMessageRawEntry struct {
	EntryID     string            `json:"entryID"`
	Event       string            `json:"event"`
	ContentType string            `json:"contentType,omitempty"`
	Metadata    map[string]string `json:"metadata"`
}

type BulkMessage struct {
	Entries  []BulkMessageEntry `json:"entries"`
	Topic    string             `json:"topic"`
	Metadata map[string]string  `json:"metadata"`
}

type BulkMessageEntry struct {
	EntryID     string                 `json:"entryID"`
	Event       map[string]interface{} `json:"event"`
	ContentType string                 `json:"contentType,omitempty"`
	Metadata    map[string]string      `json:"metadata"`
}

type AppBulkMessageEntry struct {
	EntryID     string            `json:"entryID"`
	EventStr    string            `json:"event"`
	ContentType string            `json:"contentType,omitempty"`
	Metadata    map[string]string `json:"metadata"`
}

type BulkSubscribeResponseEntry struct {
	EntryID string `json:"entryID"`
	Status  string `json:"status"`
}

// BulkSubscribeResponse is the whole bulk subscribe response sent by app
type BulkSubscribeResponse struct {
	Statuses []BulkSubscribeResponseEntry `json:"statuses"`
}

// respondWith determines the response to return when a message
// is received.
type respondWith int

const (
	respondWithSuccess respondWith = iota
	// respond with empty json message
	respondWithEmptyJSON
	// respond with error
	respondWithError
	// respond with retry
	respondWithRetry
	// respond with invalid status
	respondWithInvalidStatus
	// respond with success for all messages in bulk
	respondWithSuccessBulk
)

var (
	// using sets to make the test idempotent on multiple delivery of same message
	// receivedMessagesSubRaw       sets.String
	// receivedMessagesSubCE        sets.String
	receivedMessagesBulkRaw sets.String
	receivedMessagesBulkCE  sets.String
	desiredResponse         respondWith
	lock                    sync.Mutex
)

// indexHandler is the handler for root path
func indexHandler(w http.ResponseWriter, _ *http.Request) {
	log.Printf("indexHandler called")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(appResponse{Message: "OK"})
}

// this handles /dapr/subscribe, which is called from dapr into this app.
// this returns the list of topics the app is subscribed to.
func configureSubscribeHandler(w http.ResponseWriter, _ *http.Request) {
	t := []subscription{
		// {
		// 	PubsubName: pubsubName,
		// 	Topic:      pubsubRawSubTopic,
		// 	Route:      pubsubRawSubTopic,
		// 	Metadata: map[string]string{
		// 		"rawPayload": "true",
		// 	},
		// },
		// {
		// 	PubsubName: pubsubName,
		// 	Topic:      pubsubCESubTopic,
		// 	Route:      pubsubCESubTopic,
		// },
		{
			PubsubName: pubsubkafkaName,
			Topic:      pubsubRawBulkSubTopic,
			Route:      pubsubRawBulkSubTopic,
			Metadata: map[string]string{
				"bulkSubscribe":                    "true",
				"rawPayload":                       "true",
				"maxBulkSubCount":                  "60",
				"maxBulkAwaitDurationMilliSeconds": "1000",
			},
		},
		{
			PubsubName: pubsubkafkaName,
			Topic:      pubsubCEBulkSubTopic,
			Route:      pubsubCEBulkSubTopic,
			Metadata: map[string]string{
				"bulkSubscribe":                    "true",
				"maxBulkSubCount":                  "60",
				"maxBulkAwaitDurationMilliSeconds": "1000",
			},
		},
	}

	log.Printf("configureSubscribeHandler called; subscribing to: %v\n", t)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(t)
}

func readBulkMessageBody(reqID string, r *http.Request) (msgs []AppBulkMessageEntry, err error) {
	defer r.Body.Close()

	var body []byte
	if r.Body != nil {
		var data []byte
		data, err = io.ReadAll(r.Body)
		if err == nil {
			body = data
		}
	} else {
		// error
		err = errors.New("r.Body is nil")
	}

	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(r.URL.String(), pubsubRawBulkSubTopic) {
		msgs, err = extractBulkMessage(reqID, body, true)
		if err != nil {
			return nil, fmt.Errorf("error from extractBulkMessage: %w", err)
		}

	} else {
		msgs, err = extractBulkMessage(reqID, body, false)
		if err != nil {
			return nil, fmt.Errorf("error from extractBulkMessage: %w", err)
		}
	}
	return msgs, nil
}

func bulkSubscribeHandler(w http.ResponseWriter, r *http.Request) {
	reqID, ok := r.Context().Value("reqid").(string)
	log.Printf("(%s) bulkSubscribeHandler called %s.", reqID, r.URL)
	if reqID == "" || !ok {
		reqID = uuid.New().String()
	}

	msgs, err := readBulkMessageBody(reqID, r)

	bulkResponseEntries := make([]BulkSubscribeResponseEntry, len(msgs))

	if err != nil {
		log.Printf("(%s) Responding with DROP due to error: %v", reqID, err)
		// Return 200 with DROP status to drop message
		w.WriteHeader(http.StatusOK)
		for i, msg := range msgs {
			entryResponse := BulkSubscribeResponseEntry{}
			entryResponse.EntryID = msg.EntryID
			entryResponse.Status = "DROP"
			bulkResponseEntries[i] = entryResponse
		}
		json.NewEncoder(w).Encode(BulkSubscribeResponse{
			Statuses: bulkResponseEntries,
		})
		return
	}

	// Before we handle the error, see if we need to respond in another way
	// We still want the message so we can log it
	// if strings.HasSuffix(r.URL.String(), pubsubRawBulkSubTopic) {
	// 	krawLock.Lock()
	// 	defer krawLock.Unlock()
	// } else {
	// 	kceLock.Lock()
	// 	defer kceLock.Unlock()
	// }
	for i, msg := range msgs {
		entryResponse := BulkSubscribeResponseEntry{}
		log.Printf("(%s) bulkSubscribeHandler called %s.Index: %d, Message: %s", reqID, r.URL, i, msg)
		// switch desiredResponse {
		// case respondWithRetry:
		// 	log.Printf("(%s) Responding with RETRY for entryID %s", reqID, msg.EntryID)
		// 	entryResponse.EntryID = msg.EntryID
		// 	entryResponse.Status = "RETRY"
		// 	bulkResponseEntries[i] = entryResponse
		// 	continue
		// case respondWithSuccessBulk:
		log.Printf("(%s) Responding with SUCCESS for entryID %s", reqID, msg.EntryID)
		entryResponse.EntryID = msg.EntryID
		entryResponse.Status = "SUCCESS"
		bulkResponseEntries[i] = entryResponse

		if strings.HasSuffix(r.URL.String(), pubsubRawBulkSubTopic) && !receivedMessagesBulkRaw.Has(msg.EventStr) {
			receivedMessagesBulkRaw.Insert(msg.EventStr)
		} else if strings.HasSuffix(r.URL.String(), pubsubCEBulkSubTopic) && !receivedMessagesBulkCE.Has(msg.EventStr) {
			receivedMessagesBulkCE.Insert(msg.EventStr)
		} else {
			// This case is triggered when there is multiple redelivery of same message or a message
			// is thre for an unknown URL path

			errorMessage := fmt.Sprintf("Unexpected/Multiple redelivery of message during bulk susbcribe from %s", r.URL.String())
			log.Printf("(%s) Responding with DROP during bulk subscribe. %s", reqID, errorMessage)
			entryResponse.Status = "DROP"
		}
		// continue
		// }
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("(%s) Responding with SUCCESS", reqID)
	json.NewEncoder(w).Encode(BulkSubscribeResponse{
		Statuses: bulkResponseEntries,
	})
}

func unique(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func extractBulkMessage(reqID string, body []byte, isRawPayload bool) ([]AppBulkMessageEntry, error) {
	log.Printf("(%s) extractBulkMessage() called with body=%s", reqID, string(body))

	if !isRawPayload {
		var bulkMsg BulkMessage
		err := json.Unmarshal(body, &bulkMsg)
		if err != nil {
			log.Printf("(%s) Could not unmarshal bulkMsg: %v", reqID, err)
			return nil, err
		}

		finalMsgs := make([]AppBulkMessageEntry, len(bulkMsg.Entries))
		for i, entry := range bulkMsg.Entries {
			entryCEData := entry.Event["data"].(string)
			appMsg := AppBulkMessageEntry{
				EntryID:  entry.EntryID,
				EventStr: entryCEData,
			}
			finalMsgs[i] = appMsg
			log.Printf("(%s) output at index: %d, entry id:'%s' is: '%s':", reqID, i, entry.EntryID, entryCEData)
		}
		return finalMsgs, nil
	}
	var bulkMsg BulkRawMessage
	err := json.Unmarshal(body, &bulkMsg)
	if err != nil {
		log.Printf("(%s) Could not unmarshal raw bulkMsg: %v", reqID, err)
		return nil, err
	}

	finalMsgs := make([]AppBulkMessageEntry, len(bulkMsg.Entries))
	for i, entry := range bulkMsg.Entries {
		entryData, err := base64.StdEncoding.DecodeString(entry.Event)

		if err != nil {
			log.Printf("(%s) Could not base64 decode in bulk entry: %v", reqID, err)
			continue
		}

		entryDataStr := string(entryData)
		log.Printf("(%s) output from base64 in bulk entry %s is:'%s'", reqID, entry.EntryID, entryDataStr)

		var actualMsg string
		err = json.Unmarshal([]byte(entryDataStr), &actualMsg)
		if err != nil {
			// Log only
			log.Printf("(%s) Error extracing JSON from raw event in bulk entry %s is: %v", reqID, entry.EntryID, err)
		} else {
			log.Printf("(%s) Output of JSON from raw event in bulk entry %s is: %v", reqID, entry.EntryID, actualMsg)
			entryDataStr = actualMsg
		}

		appMsg := AppBulkMessageEntry{
			EntryID:  entry.EntryID,
			EventStr: entryDataStr,
		}
		finalMsgs[i] = appMsg
		log.Printf("(%s) output at index: %d, entry id:'%s' is: '%s':", reqID, i, entry.EntryID, entryData)
	}
	return finalMsgs, nil
}

// the test calls this to get the messages received
func getReceivedMessages(w http.ResponseWriter, r *http.Request) {
	reqID, ok := r.Context().Value("reqid").(string)
	if reqID == "" || !ok {
		reqID = "s-" + uuid.New().String()
	}

	response := receivedMessagesResponse{
		// ReceivedByTopicRawSub:     unique(receivedMessagesSubRaw.List()),
		// ReceivedByTopicCESub:      unique(receivedMessagesSubCE.List()),
		ReceivedByTopicRawBulkSub: unique(receivedMessagesBulkRaw.List()),
		ReceivedByTopicCEBulkSub:  unique(receivedMessagesBulkCE.List()),
	}

	log.Printf("getReceivedMessages called. reqID=%s response=%s", reqID, response)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// setDesiredResponse returns an http.HandlerFunc that sets the desired response
// to `resp` and logs `msg`.
func setDesiredResponse(resp respondWith, msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		lock.Lock()
		defer lock.Unlock()
		log.Print(msg)
		desiredResponse = resp
		w.WriteHeader(http.StatusOK)
	}
}

// handler called for empty-json case.
func initializeHandler(w http.ResponseWriter, _ *http.Request) {
	initializeSets()
	w.WriteHeader(http.StatusOK)
}

// initialize all the sets for a clean test.
func initializeSets() {
	// initialize all the sets
	// receivedMessagesSubRaw = sets.NewString()
	// receivedMessagesSubCE = sets.NewString()
	receivedMessagesBulkRaw = sets.NewString()
	receivedMessagesBulkCE = sets.NewString()
}

// appRouter initializes restful api router
func appRouter() *mux.Router {
	log.Printf("Called appRouter()")
	router := mux.NewRouter().StrictSlash(true)

	// Log requests and their processing time
	router.Use(utils.LoggerMiddleware)

	router.HandleFunc("/", indexHandler).Methods("GET")

	router.HandleFunc("/getMessages", getReceivedMessages).Methods("POST")
	router.HandleFunc("/set-respond-success",
		setDesiredResponse(respondWithSuccess, "set respond with success")).Methods("POST")
	router.HandleFunc("/set-respond-success-bulk",
		setDesiredResponse(respondWithSuccessBulk, "set respond with success for bulk")).Methods("POST")
	router.HandleFunc("/set-respond-error",
		setDesiredResponse(respondWithError, "set respond with error")).Methods("POST")
	router.HandleFunc("/set-respond-retry",
		setDesiredResponse(respondWithRetry, "set respond with retry")).Methods("POST")
	router.HandleFunc("/set-respond-empty-json",
		setDesiredResponse(respondWithEmptyJSON, "set respond with empty json"))
	router.HandleFunc("/set-respond-invalid-status",
		setDesiredResponse(respondWithInvalidStatus, "set respond with invalid status")).Methods("POST")
	router.HandleFunc("/initialize", initializeHandler).Methods("POST")

	router.HandleFunc("/dapr/subscribe", configureSubscribeHandler).Methods("GET")

	// router.HandleFunc("/"+pubsubRawSubTopic, subscribeHandler).Methods("POST")
	// router.HandleFunc("/"+pubsubCESubTopic, subscribeHandler).Methods("POST")
	router.HandleFunc("/"+pubsubRawBulkSubTopic, bulkSubscribeHandler).Methods("POST")
	router.HandleFunc("/"+pubsubCEBulkSubTopic, bulkSubscribeHandler).Methods("POST")

	router.Use(mux.CORSMethodMiddleware(router))

	return router
}

func main() {
	// initialize sets on application start
	initializeSets()

	log.Printf("Dapr E2E test app: pubsub - listening on http://localhost:%d", appPort)
	utils.StartServer(appPort, appRouter, true, false)
}