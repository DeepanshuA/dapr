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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/apimachinery/pkg/util/sets"

	commonv1pb "github.com/dapr/dapr/pkg/proto/common/v1"
	runtimev1pb "github.com/dapr/dapr/pkg/proto/runtime/v1"
)

const (
	appPort               = "3000"
	pubsubA               = "pubsub-a-topic-grpc"
	pubsubB               = "pubsub-b-topic-grpc"
	pubsubC               = "pubsub-c-topic-grpc"
	pubsubRaw             = "pubsub-raw-topic-grpc"
	pubsubBulkTopic       = "pubsub-bulk-topic-grpc"
	pubsubRawBulkTopic    = "pubsub-raw-bulk-topic-grpc"
	pubsubCEBulkTopic     = "pubsub-ce-bulk-topic-grpc"
	pubsubDefBulkTopic    = "pubsub-def-bulk-topic-grpc"
	pubsubRawSubTopic     = "pubsub-raw-sub-topic-grpc"
	pubsubCESubTopic      = "pubsub-ce-sub-topic-grpc"
	pubsubRawBulkSubTopic = "pubsub-raw-bulk-sub-topic-grpc"
	pubsubCEBulkSubTopic  = "pubsub-ce-bulk-sub-topic-grpc"
	pubsubName            = "messagebus"
	pubsubKafka           = "kafka-messagebus"
)

var (
	// using sets to make the test idempotent on multiple delivery of same message.
	receivedMessagesA            sets.String
	receivedMessagesB            sets.String
	receivedMessagesC            sets.String
	receivedMessagesRaw          sets.String
	receivedMessagesBulkTopic    sets.String
	receivedMessagesRawBulkTopic sets.String
	receivedMessagesCEBulkTopic  sets.String
	receivedMessagesDefBulkTopic sets.String
	receivedMessagesSubRaw       sets.String
	receivedMessagesSubCE        sets.String
	receivedMessagesRawBulkSub   sets.String
	receivedMessagesCEBulkSub    sets.String

	// boolean variable to respond with empty json message if set.
	respondWithEmptyJSON bool
	// boolean variable to respond with error if set.
	respondWithError bool
	// boolean variable to respond with retry if set.
	respondWithRetry bool
	// boolean variable to respond with invalid status if set.
	respondWithInvalidStatus bool
	lock                     sync.Mutex
)

type receivedMessagesResponse struct {
	ReceivedByTopicA          []string `json:"pubsub-a-topic"`
	ReceivedByTopicB          []string `json:"pubsub-b-topic"`
	ReceivedByTopicC          []string `json:"pubsub-c-topic"`
	ReceivedByTopicRaw        []string `json:"pubsub-raw-topic"`
	ReceivedByTopicBulk       []string `json:"pubsub-bulk-topic"`
	ReceivedByTopicRawBulk    []string `json:"pubsub-raw-bulk-topic"`
	ReceivedByTopicCEBulk     []string `json:"pubsub-ce-bulk-topic"`
	ReceivedByTopicDefBulk    []string `json:"pubsub-def-bulk-topic"`
	ReceivedByTopicRawSub     []string `json:"pubsub-raw-sub-topic"`
	ReceivedByTopicCESub      []string `json:"pubsub-ce-sub-topic"`
	ReceivedByTopicRawBulkSub []string `json:"pubsub-raw-bulk-sub-topic"`
	ReceivedByTopicCEBulkSub  []string `json:"pubsub-ce-bulk-sub-topic"`
}

// server is our user app.
type server struct{}

func main() {
	log.Printf("Initializing grpc")

	/* #nosec */
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", appPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	lock.Lock()
	initializeSets()
	lock.Unlock()

	/* #nosec */
	s := grpc.NewServer()
	runtimev1pb.RegisterAppCallbackServer(s, &server{})
	runtimev1pb.RegisterAppCallbackAlphaServer(s, &server{})

	log.Println("Client starting...")

	// Stop the gRPC server when we get a termination signal
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT) //nolint:staticcheck
	go func() {
		// Wait for cancelation signal
		<-stopCh
		log.Println("Shutdown signal received")
		s.GracefulStop()
	}()

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	log.Println("App shut down")
}

func initializeSets() {
	receivedMessagesA = sets.NewString()
	receivedMessagesB = sets.NewString()
	receivedMessagesC = sets.NewString()
	receivedMessagesRaw = sets.NewString()
	receivedMessagesBulkTopic = sets.NewString()
	receivedMessagesRawBulkTopic = sets.NewString()
	receivedMessagesCEBulkTopic = sets.NewString()
	receivedMessagesDefBulkTopic = sets.NewString()
	receivedMessagesSubRaw = sets.NewString()
	receivedMessagesSubCE = sets.NewString()
	receivedMessagesRawBulkSub = sets.NewString()
	receivedMessagesCEBulkSub = sets.NewString()
}

// This method gets invoked when a remote service has called the app through Dapr
// The payload carries a Method to identify the method, a set of metadata properties and an optional payload.
func (s *server) OnInvoke(ctx context.Context, in *commonv1pb.InvokeRequest) (*commonv1pb.InvokeResponse, error) {
	reqID := "s-" + uuid.New().String()
	if in.HttpExtension != nil && in.HttpExtension.Querystring != "" {
		qs, err := url.ParseQuery(in.HttpExtension.Querystring)
		if err == nil && qs.Has("reqid") {
			reqID = qs.Get("reqid")
		}
	}

	log.Printf("(%s) Got invoked method %s", reqID, in.Method)

	lock.Lock()
	defer lock.Unlock()

	respBody := &anypb.Any{}
	switch in.Method {
	case "getMessages":
		respBody.Value = s.getMessages(reqID)
	case "initialize":
		initializeSets()
	case "set-respond-error":
		s.setRespondWithError()
	case "set-respond-retry":
		s.setRespondWithRetry()
	case "set-respond-empty-json":
		s.setRespondWithEmptyJSON()
	case "set-respond-invalid-status":
		s.setRespondWithInvalidStatus()
	}

	return &commonv1pb.InvokeResponse{Data: respBody, ContentType: "application/json"}, nil
}

func (s *server) getMessages(reqID string) []byte {
	resp := receivedMessagesResponse{
		ReceivedByTopicA:          receivedMessagesA.List(),
		ReceivedByTopicB:          receivedMessagesB.List(),
		ReceivedByTopicC:          receivedMessagesC.List(),
		ReceivedByTopicRaw:        receivedMessagesRaw.List(),
		ReceivedByTopicBulk:       receivedMessagesBulkTopic.List(),
		ReceivedByTopicRawBulk:    receivedMessagesRawBulkTopic.List(),
		ReceivedByTopicCEBulk:     receivedMessagesCEBulkTopic.List(),
		ReceivedByTopicDefBulk:    receivedMessagesDefBulkTopic.List(),
		ReceivedByTopicRawSub:     receivedMessagesSubRaw.List(),
		ReceivedByTopicCESub:      receivedMessagesSubCE.List(),
		ReceivedByTopicRawBulkSub: receivedMessagesRawBulkSub.List(),
		ReceivedByTopicCEBulkSub:  receivedMessagesCEBulkSub.List(),
	}

	rawResp, _ := json.Marshal(resp)
	log.Printf("(%s) getMessages response: %s", reqID, string(rawResp))
	return rawResp
}

func (s *server) setRespondWithError() {
	log.Println("setRespondWithError called")
	respondWithError = true
}

func (s *server) setRespondWithRetry() {
	log.Println("setRespondWithRetry called")
	respondWithRetry = true
}

func (s *server) setRespondWithEmptyJSON() {
	log.Println("setRespondWithEmptyJSON called")
	respondWithEmptyJSON = true
}

func (s *server) setRespondWithInvalidStatus() {
	log.Println("setRespondWithInvalidStatus called")
	respondWithInvalidStatus = true
}

// Dapr will call this method to get the list of topics the app wants to subscribe to. In this example, we are telling Dapr
// to subscribe to a topic named TopicA.
func (s *server) ListTopicSubscriptions(ctx context.Context, in *emptypb.Empty) (*runtimev1pb.ListTopicSubscriptionsResponse, error) {
	log.Println("List Topic Subscription called")
	resp := &runtimev1pb.ListTopicSubscriptionsResponse{
		Subscriptions: []*runtimev1pb.TopicSubscription{
			{
				PubsubName: pubsubName,
				Topic:      pubsubA,
			},
			{
				PubsubName: pubsubName,
				Topic:      pubsubB,
			},
			// pubsub-c-topic-grpc is loaded from the YAML/CRD
			// tests/config/app_topic_subscription_pubsub_grpc.yaml.
			{
				PubsubName: pubsubName,
				Topic:      pubsubRaw,
				Metadata: map[string]string{
					"rawPayload": "true",
				},
			},
			// all bulk topics are passed through Kafka which has the bulk publish implemented
			{
				PubsubName: pubsubKafka,
				Topic:      pubsubBulkTopic,
			},
			{
				PubsubName: pubsubKafka,
				Topic:      pubsubRawBulkTopic,
				Metadata: map[string]string{
					"rawPayload": "true",
				},
			},
			{
				PubsubName: pubsubKafka,
				Topic:      pubsubCEBulkTopic,
			},
			// bulk publish default impl goes to redis as that does not have bulk publish implemented
			{
				PubsubName: pubsubName,
				Topic:      pubsubDefBulkTopic,
			},
			{
				PubsubName: pubsubKafka,
				Topic:      pubsubRawSubTopic,
				Routes: &runtimev1pb.TopicRoutes{
					Default: pubsubRawSubTopic,
				},
				Metadata: map[string]string{
					"rawPayload": "true",
				},
			},
			{
				PubsubName: pubsubKafka,
				Topic:      pubsubCESubTopic,
				Routes: &runtimev1pb.TopicRoutes{
					Default: pubsubCESubTopic,
				},
			},
			{
				PubsubName: pubsubKafka,
				Topic:      pubsubRawBulkSubTopic,
				Routes: &runtimev1pb.TopicRoutes{
					Default: pubsubRawBulkSubTopic,
				},
				Metadata: map[string]string{
					"rawPayload":                "true",
					"bulkSubscribe":             "true",
					"maxBulkSubCount":           "60",
					"maxBulkSubAwaitDurationMs": "1000",
				},
			},
			{
				PubsubName: pubsubKafka,
				Topic:      pubsubCEBulkSubTopic,
				Routes: &runtimev1pb.TopicRoutes{
					Default: pubsubCEBulkSubTopic,
				},
				Metadata: map[string]string{
					"bulkSubscribe":             "true",
					"maxBulkSubCount":           "60",
					"maxBulkSubAwaitDurationMs": "1000",
				},
			},
		},
	}
	log.Printf("Returning topic subscriptions as %v", resp)
	return resp, nil
}

// This method is fired whenever a message has been published to a topic that has been subscribed.
// Dapr sends published messages in a CloudEvents 1.0 envelope.
func (s *server) OnTopicEvent(ctx context.Context, in *runtimev1pb.TopicEventRequest) (*runtimev1pb.TopicEventResponse, error) {
	lock.Lock()
	defer lock.Unlock()

	reqID := uuid.New().String()
	log.Printf("(%s) Message arrived - Topic: %s, Message: %s", reqID, in.Topic, string(in.Data))

	if respondWithRetry {
		log.Printf("(%s) Responding with RETRY", reqID)
		return &runtimev1pb.TopicEventResponse{
			Status: runtimev1pb.TopicEventResponse_RETRY, //nolint:nosnakecase
		}, nil
	} else if respondWithError {
		log.Printf("(%s) Responding with ERROR", reqID)
		// do not store received messages, respond with error
		return nil, errors.New("error response")
	} else if respondWithInvalidStatus {
		log.Printf("(%s) Responding with INVALID", reqID)
		// do not store received messages, respond with success but an invalid status
		return &runtimev1pb.TopicEventResponse{
			Status: 4,
		}, nil
	}

	if in.Data == nil {
		log.Printf("(%s) Responding with DROP. in.Data is nil", reqID)
		// Return success with DROP status to drop message
		return &runtimev1pb.TopicEventResponse{
			Status: runtimev1pb.TopicEventResponse_DROP, //nolint:nosnakecase
		}, nil
	}
	log.Printf("(%s) data %s and the content type (%s)", reqID, in.Data, in.DataContentType)
	var msg string
	var err error
	if !strings.Contains(in.Topic, "bulk") {
		// This is the old flow where always the content type is application/json
		// and data is always json serialized
		err = json.Unmarshal(in.Data, &msg)
		if err != nil {
			log.Printf("(%s) Responding with DROP. Error while unmarshaling JSON data: %v", reqID, err)
			// Return success with DROP status to drop message
			return &runtimev1pb.TopicEventResponse{
				Status: runtimev1pb.TopicEventResponse_DROP, //nolint:nosnakecase
			}, err
		}
		if strings.HasPrefix(in.Topic, pubsubRaw) {
			var actualMsg string
			err = json.Unmarshal([]byte(msg), &actualMsg)
			if err != nil {
				log.Printf("(%s) Error extracing JSON from raw event: %v", reqID, err)
			} else {
				msg = actualMsg
			}
		}
	} else if strings.Contains(in.Topic, "bulk") {
		// In bulk publish data and data content type match is important and
		// enforced/expected
		if in.DataContentType == "application/json" || in.DataContentType == "application/cloudevents+json" {
			err = json.Unmarshal(in.Data, &msg)
			if err != nil {
				log.Printf("(%s) Responding with DROP. Error while unmarshaling JSON data: %v", reqID, err)
				// Return success with DROP status to drop message
				return &runtimev1pb.TopicEventResponse{
					Status: runtimev1pb.TopicEventResponse_DROP, //nolint:nosnakecase
				}, err
			}
		} else if strings.HasPrefix(in.DataContentType, "text/") {
			msg = (string)(in.Data)
		} else if strings.Contains(in.Topic, "raw") {
			// All raw payload topics are assumed to have "raw" in the name
			// this is for the bulk case
			// This is simply for E2E only ....
			// we are assuming raw payload is also a string here .... In general msg should be []byte only and compared as []byte
			// raw payload for bulk is set from a string so this scenario holds true
			msg = string(in.Data)
		}
	}

	log.Printf("(%s) Received message: %s - %s", reqID, in.Topic, msg)

	if strings.HasPrefix(in.Topic, pubsubA) && !receivedMessagesA.Has(msg) {
		receivedMessagesA.Insert(msg)
	} else if strings.HasPrefix(in.Topic, pubsubB) && !receivedMessagesB.Has(msg) {
		receivedMessagesB.Insert(msg)
	} else if strings.HasPrefix(in.Topic, pubsubC) && !receivedMessagesC.Has(msg) {
		receivedMessagesC.Insert(msg)
	} else if strings.HasPrefix(in.Topic, pubsubRaw) && !receivedMessagesRaw.Has(msg) {
		receivedMessagesRaw.Insert(msg)
	} else if strings.HasSuffix(in.Topic, pubsubBulkTopic) && !receivedMessagesBulkTopic.Has(msg) {
		receivedMessagesBulkTopic.Insert(msg)
	} else if strings.HasSuffix(in.Topic, pubsubRawBulkTopic) && !receivedMessagesRawBulkTopic.Has(msg) {
		receivedMessagesRawBulkTopic.Insert(msg)
	} else if strings.HasSuffix(in.Topic, pubsubCEBulkTopic) && !receivedMessagesCEBulkTopic.Has(msg) {
		receivedMessagesCEBulkTopic.Insert(msg)
	} else if strings.HasSuffix(in.Topic, pubsubDefBulkTopic) && !receivedMessagesDefBulkTopic.Has(msg) {
		receivedMessagesDefBulkTopic.Insert(msg)
	} else if strings.HasPrefix(in.Topic, pubsubRawSubTopic) && !receivedMessagesSubRaw.Has(msg) {
		receivedMessagesSubRaw.Insert(msg)
	} else if strings.HasPrefix(in.Topic, pubsubCESubTopic) && !receivedMessagesSubCE.Has(msg) {
		receivedMessagesSubCE.Insert(msg)
	} else {
		log.Printf("(%s) Received duplicate message: %s - %s", reqID, in.Topic, msg)
	}

	if respondWithEmptyJSON {
		log.Printf("(%s) Responding with {}", reqID)
		return &runtimev1pb.TopicEventResponse{}, nil
	}

	log.Printf("(%s) Responding with SUCCESS", reqID)
	return &runtimev1pb.TopicEventResponse{
		Status: runtimev1pb.TopicEventResponse_SUCCESS, //nolint:nosnakecase
	}, nil
}

func (s *server) OnBulkTopicEventAlpha1(ctx context.Context, in *runtimev1pb.TopicEventBulkRequest) (*runtimev1pb.TopicEventBulkResponse, error) {
	reqID := uuid.New().String()
	log.Printf("(%s) Entered in OnBulkTopicEventAlpha1 in Bulk Subscribe - Topic: %s", reqID, in.Topic)
	lock.Lock()
	defer lock.Unlock()

	bulkResponses := make([]*runtimev1pb.TopicEventBulkResponseEntry, len(in.Entries))

	for i, entry := range in.Entries {
		if entry.Event == nil {
			log.Printf("(%s) Responding with DROP in bulk subscribe for entryId: %s. entry.Event is nil", reqID, entry.EntryId)
			// Return success with DROP status to drop message
			bulkResponses[i] = &runtimev1pb.TopicEventBulkResponseEntry{
				EntryId: entry.EntryId,
				Status:  runtimev1pb.TopicEventResponse_DROP, //nolint:nosnakecase
			}
		}
		var msg string
		if strings.HasPrefix(in.Topic, pubsubCEBulkSubTopic) {
			log.Printf("(%s) Message arrived in Bulk Subscribe - Topic: %s, Message: %s", reqID, in.Topic, string(entry.GetCloudEvent().Data))
			// var ceMsg map[string]interface{}
			// err := json.Unmarshal(entry.GetCloudEvent().Data, &ceMsg)
			// if err != nil {
			// 	log.Printf("(%s) Error extracing ce event in bulk subscribe for entryId: %s: %v", reqID, entry.EntryId, err)
			// 	bulkResponses[i] = &runtimev1pb.TopicEventBulkResponseEntry{
			// 		EntryId: entry.EntryId,
			// 		Status:  runtimev1pb.TopicEventResponse_DROP, //nolint:nosnakecase
			// 	}
			// 	continue
			// }
			msg = string(entry.GetCloudEvent().Data) //ceMsg["data"].(string)
			log.Printf("(%s) Value of ce event in bulk subscribe for entryId: %s: %s", reqID, entry.EntryId, msg)
		} else {
			log.Printf("(%s) Message arrived in Bulk Subscribe - Topic: %s, Message: %s", reqID, in.Topic, string(entry.GetBytes()))
			err := json.Unmarshal(entry.GetBytes(), &msg)
			if err != nil {
				log.Printf("(%s) Error extracing raw event in bulk subscribe for entryId: %s: %v", reqID, entry.EntryId, err)
				// Return success with DROP status to drop message
				bulkResponses[i] = &runtimev1pb.TopicEventBulkResponseEntry{
					EntryId: entry.EntryId,
					Status:  runtimev1pb.TopicEventResponse_DROP, //nolint:nosnakecase
				}
				continue
			}
			log.Printf("(%s) Value of raw event in bulk subscribe for entryId: %s: %s", reqID, entry.EntryId, msg)
		}

		bulkResponses[i] = &runtimev1pb.TopicEventBulkResponseEntry{
			EntryId: entry.EntryId,
			Status:  runtimev1pb.TopicEventResponse_SUCCESS, //nolint:nosnakecase
		}
		if strings.HasPrefix(in.Topic, pubsubRawBulkSubTopic) && !receivedMessagesRawBulkSub.Has(msg) {
			receivedMessagesRawBulkSub.Insert(msg)
		} else if strings.HasPrefix(in.Topic, pubsubCEBulkSubTopic) && !receivedMessagesCEBulkSub.Has(msg) {
			receivedMessagesCEBulkSub.Insert(msg)
		} else {
			log.Printf("(%s) Received duplicate message in bulk subscribe: %s - %s", reqID, in.Topic, msg)
		}
	}
	log.Printf("(%s) Responding with SUCCESS for bulk subscribe", reqID)
	return &runtimev1pb.TopicEventBulkResponse{
		Statuses: bulkResponses,
	}, nil
}

// Dapr will call this method to get the list of bindings the app will get invoked by. In this example, we are telling Dapr
// To invoke our app with a binding named storage.
func (s *server) ListInputBindings(ctx context.Context, in *emptypb.Empty) (*runtimev1pb.ListInputBindingsResponse, error) {
	log.Println("List Input Bindings called")
	return &runtimev1pb.ListInputBindingsResponse{}, nil
}

// This method gets invoked every time a new event is fired from a registered binding. The message carries the binding name, a payload and optional metadata.
func (s *server) OnBindingEvent(ctx context.Context, in *runtimev1pb.BindingEventRequest) (*runtimev1pb.BindingEventResponse, error) {
	log.Printf("Invoked from binding: %s", in.Name)
	return &runtimev1pb.BindingEventResponse{}, nil
}
