package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/dapr/components-contrib/pubsub"
	componentsV1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	channelt "github.com/dapr/dapr/pkg/channel/testing"
	invokev1 "github.com/dapr/dapr/pkg/messaging/v1"
	"github.com/dapr/dapr/pkg/modes"
	commonV1 "github.com/dapr/dapr/pkg/proto/common/v1"
	runtimev1pb "github.com/dapr/dapr/pkg/proto/runtime/v1"
	runtimePubsub "github.com/dapr/dapr/pkg/runtime/pubsub"
	"github.com/dapr/kit/logger"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	order1  string = `{"data":{"orderId":"1"},"datacontenttype":"application/json","id":"9b6767c3-04b5-4871-96ae-c6bde0d5e16d","pubsubname":"orderpubsub","source":"checkout","specversion":"1.0","topic":"orders","traceid":"00-e61de949bb4de415a7af49fc86675648-ffb64972bb907224-01","traceparent":"00-e61de949bb4de415a7af49fc86675648-ffb64972bb907224-01","tracestate":"","type":"type1"}`
	order2  string = `{"data":{"orderId":"2"},"datacontenttype":"application/json","id":"993f4e4a-05e5-4772-94a4-e899b1af0131","pubsubname":"orderpubsub","source":"checkout","specversion":"1.0","topic":"orders","traceid":"00-1343b02c3af4f9b352d4cb83d6c8cb81-82a64f8c4433e2c4-01","traceparent":"00-1343b02c3af4f9b352d4cb83d6c8cb81-82a64f8c4433e2c4-01","tracestate":"","type":"type2"}`
	order3  string = `{"data":{"orderId":"3"},"datacontenttype":"application/json","id":"6767010u-04b5-4871-96ae-c6bde0d5e16d","pubsubname":"orderpubsub","source":"checkout","specversion":"1.0","topic":"orders","traceid":"00-e61de949bb4de415a7af49fc86675648-ffb64972bb907224-01","traceparent":"00-e61de949bb4de415a7af49fc86675648-ffb64972bb907224-01","tracestate":"","type":"type1"}`
	order4  string = `{"data":{"orderId":"4"},"datacontenttype":"application/json","id":"91011121-05e5-4772-94a4-e899b1af0131","pubsubname":"orderpubsub","source":"checkout","specversion":"1.0","topic":"orders","traceid":"00-1343b02c3af4f9b352d4cb83d6c8cb81-82a64f8c4433e2c4-01","traceparent":"00-1343b02c3af4f9b352d4cb83d6c8cb81-82a64f8c4433e2c4-01","tracestate":"","type":"type2"}`
	order5  string = `{"data":{"orderId":"5"},"datacontenttype":"application/json","id":"718271cd-04b5-4871-96ae-c6bde0d5e16d","pubsubname":"orderpubsub","source":"checkout","specversion":"1.0","topic":"orders","traceid":"00-e61de949bb4de415a7af49fc86675648-ffb64972bb907224-01","traceparent":"00-e61de949bb4de415a7af49fc86675648-ffb64972bb907224-01","tracestate":"","type":"type1"}`
	order6  string = `{"data":{"orderId":"6"},"datacontenttype":"application/json","id":"7uw2233d-05e5-4772-94a4-e899b1af0131","pubsubname":"orderpubsub","source":"checkout","specversion":"1.0","topic":"orders","traceid":"00-1343b02c3af4f9b352d4cb83d6c8cb81-82a64f8c4433e2c4-01","traceparent":"00-1343b02c3af4f9b352d4cb83d6c8cb81-82a64f8c4433e2c4-01","tracestate":"","type":"type2"}`
	order7  string = `{"data":{"orderId":"7"},"datacontenttype":"application/json","id":"78sqs98s-04b5-4871-96ae-c6bde0d5e16d","pubsubname":"orderpubsub","source":"checkout","specversion":"1.0","topic":"orders","traceid":"00-e61de949bb4de415a7af49fc86675648-ffb64972bb907224-01","traceparent":"00-e61de949bb4de415a7af49fc86675648-ffb64972bb907224-01","tracestate":"","type":"type1"}`
	order8  string = `{"data":{"orderId":"8"},"datacontenttype":"application/json","id":"45122j82-05e5-4772-94a4-e899b1af0131","pubsubname":"orderpubsub","source":"checkout","specversion":"1.0","topic":"orders","traceid":"00-1343b02c3af4f9b352d4cb83d6c8cb81-82a64f8c4433e2c4-01","traceparent":"00-1343b02c3af4f9b352d4cb83d6c8cb81-82a64f8c4433e2c4-01","tracestate":"","type":"type1"}`
	order9  string = `{"orderId":"9","type":"type1"}`
	order10 string = `{"data":{"orderId":"10"},"datacontenttype":"application/json","id":"ded2rd44-05e5-4772-94a4-e899b1af0131","pubsubname":"orderpubsub","source":"checkout","specversion":"1.0","topic":"orders","traceid":"00-1343b02c3af4f9b352d4cb83d6c8cb81-82a64f8c4433e2c4-01","traceparent":"00-1343b02c3af4f9b352d4cb83d6c8cb81-82a64f8c4433e2c4-01","tracestate":"","type":"type2"}`
)

func getBulkMessageEntries(len int) []pubsub.BulkMessageEntry {
	bulkEntries := make([]pubsub.BulkMessageEntry, 10)

	bulkEntries[0] = pubsub.BulkMessageEntry{EntryID: "1111111a", Event: []byte(order1)}
	bulkEntries[1] = pubsub.BulkMessageEntry{EntryID: "2222222b", Event: []byte(order2)}
	bulkEntries[2] = pubsub.BulkMessageEntry{EntryID: "333333c", Event: []byte(order3)}
	bulkEntries[3] = pubsub.BulkMessageEntry{EntryID: "4444444d", Event: []byte(order4)}
	bulkEntries[4] = pubsub.BulkMessageEntry{EntryID: "5555555e", Event: []byte(order5)}
	bulkEntries[5] = pubsub.BulkMessageEntry{EntryID: "66666666f", Event: []byte(order6)}
	bulkEntries[6] = pubsub.BulkMessageEntry{EntryID: "7777777g", Event: []byte(order7)}
	bulkEntries[7] = pubsub.BulkMessageEntry{EntryID: "8888888h", Event: []byte(order8)}
	bulkEntries[8] = pubsub.BulkMessageEntry{EntryID: "9999999i", Event: []byte(order9)}
	bulkEntries[9] = pubsub.BulkMessageEntry{EntryID: "10101010j", Event: []byte(order10)}

	return bulkEntries[:len]
}

func TestBulkSubscribe(t *testing.T) {
	testBulkSubscribePubsub := "bulkSubscribePubSub"
	pubsubComponent := componentsV1alpha1.Component{
		ObjectMeta: metaV1.ObjectMeta{
			Name: testBulkSubscribePubsub,
		},
		Spec: componentsV1alpha1.ComponentSpec{
			Type:     "pubsub.mockPubSub",
			Version:  "v1",
			Metadata: getFakeMetadataItems(),
		},
	}

	t.Run("bulk Subscribe Message for raw payload", func(t *testing.T) {
		rt := NewTestDaprRuntime(modes.StandaloneMode)
		defer stopRuntime(t, rt)
		rt.pubSubRegistry.RegisterComponent(
			func(_ logger.Logger) pubsub.PubSub {
				return &mockSubscribePubSub{}
			},
			"mockPubSub",
		)
		req := invokev1.NewInvokeMethodRequest("dapr/subscribe")
		req.WithHTTPExtension(http.MethodGet, "")
		req.WithRawData(nil, invokev1.JSONContentType)

		subscriptionItems := []runtimePubsub.SubscriptionJSON{
			{PubsubName: testBulkSubscribePubsub, Topic: "topic0", Route: "orders", Metadata: map[string]string{"bulkSubscribe": "true", "rawPayload": "true"}},
		}
		sub, _ := json.Marshal(subscriptionItems)
		fakeResp := invokev1.NewInvokeMethodResponse(200, "OK", nil)
		fakeResp.WithRawData(sub, "application/json")

		mockAppChannel := new(channelt.MockAppChannel)
		mockAppChannel.Init()
		rt.appChannel = mockAppChannel
		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), req).Return(fakeResp, nil)
		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), mock.Anything).Return(fakeResp, nil)

		require.NoError(t, rt.initPubSub(pubsubComponent))
		rt.startSubscriptions()

		err := rt.Publish(&pubsub.PublishRequest{
			PubsubName: testBulkSubscribePubsub,
			Topic:      "topic0",
			Data:       []byte(`{"orderId":"1"}`),
		})
		assert.Nil(t, err)
		pubsubIns := rt.pubSubs[testBulkSubscribePubsub].component.(*mockSubscribePubSub)
		assert.Equal(t, 1, pubsubIns.bulkPubCount["topic0"])
		assert.True(t, pubsubIns.isBulkSubscribe)
		reqs := mockAppChannel.GetInvokedRequest()
		mockAppChannel.AssertNumberOfCalls(t, "InvokeMethod", 2)
		assert.Contains(t, string(reqs["orders"].Message().Data.Value), "event\":\"eyJvcmRlcklkIjoiMSJ9\"")
	})

	t.Run("bulk Subscribe Message for cloud event", func(t *testing.T) {
		rt := NewTestDaprRuntime(modes.StandaloneMode)
		defer stopRuntime(t, rt)
		rt.pubSubRegistry.RegisterComponent(
			func(_ logger.Logger) pubsub.PubSub {
				return &mockSubscribePubSub{}
			},
			"mockPubSub",
		)
		req := invokev1.NewInvokeMethodRequest("dapr/subscribe")
		req.WithHTTPExtension(http.MethodGet, "")
		req.WithRawData(nil, invokev1.JSONContentType)

		subscriptionItems := []runtimePubsub.SubscriptionJSON{
			{PubsubName: testBulkSubscribePubsub, Topic: "topic0", Route: "orders", Metadata: map[string]string{"bulkSubscribe": "true"}},
		}
		sub, _ := json.Marshal(subscriptionItems)
		fakeResp := invokev1.NewInvokeMethodResponse(200, "OK", nil)
		fakeResp.WithRawData(sub, "application/json")

		mockAppChannel := new(channelt.MockAppChannel)
		mockAppChannel.Init()
		rt.appChannel = mockAppChannel
		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), req).Return(fakeResp, nil)
		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), mock.Anything).Return(fakeResp, nil)

		require.NoError(t, rt.initPubSub(pubsubComponent))
		rt.startSubscriptions()

		order := `{"data":{"orderId":1},"datacontenttype":"application/json","id":"8b540b03-04b5-4871-96ae-c6bde0d5e16d","pubsubname":"orderpubsub","source":"checkout","specversion":"1.0","topic":"orders","traceid":"00-e61de949bb4de415a7af49fc86675648-ffb64972bb907224-01","traceparent":"00-e61de949bb4de415a7af49fc86675648-ffb64972bb907224-01","tracestate":"","type":"com.dapr.event.sent"}`

		err := rt.Publish(&pubsub.PublishRequest{
			PubsubName: testBulkSubscribePubsub,
			Topic:      "topic0",
			Data:       []byte(order),
		})
		assert.Nil(t, err)
		pubsubIns := rt.pubSubs[testBulkSubscribePubsub].component.(*mockSubscribePubSub)
		assert.Equal(t, 1, pubsubIns.bulkPubCount["topic0"])
		assert.True(t, pubsubIns.isBulkSubscribe)
		reqs := mockAppChannel.GetInvokedRequest()
		mockAppChannel.AssertNumberOfCalls(t, "InvokeMethod", 2)
		assert.Contains(t, string(reqs["orders"].Message().Data.Value), "\"event\":"+order)
	})

	t.Run("bulk Subscribe multiple Messages at once for cloud events", func(t *testing.T) {
		rt := NewTestDaprRuntime(modes.StandaloneMode)
		defer stopRuntime(t, rt)
		rt.pubSubRegistry.RegisterComponent(
			func(_ logger.Logger) pubsub.PubSub {
				return &mockSubscribePubSub{}
			},
			"mockPubSub",
		)
		req := invokev1.NewInvokeMethodRequest("dapr/subscribe")
		req.WithHTTPExtension(http.MethodGet, "")
		req.WithRawData(nil, invokev1.JSONContentType)

		subscriptionItems := []runtimePubsub.SubscriptionJSON{
			{PubsubName: testBulkSubscribePubsub, Topic: "topic0", Route: "orders", Metadata: map[string]string{"bulkSubscribe": "true"}},
		}
		sub, _ := json.Marshal(subscriptionItems)
		fakeResp := invokev1.NewInvokeMethodResponse(200, "OK", nil)
		fakeResp.WithRawData(sub, "application/json")

		mockAppChannel := new(channelt.MockAppChannel)
		mockAppChannel.Init()
		rt.appChannel = mockAppChannel
		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), req).Return(fakeResp, nil)
		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), mock.Anything).Return(fakeResp, nil)

		require.NoError(t, rt.initPubSub(pubsubComponent))
		rt.startSubscriptions()

		msgArr := getBulkMessageEntries(2)

		_, err := rt.BulkPublish(context.TODO(), &pubsub.BulkPublishRequest{
			PubsubName: testBulkSubscribePubsub,
			Topic:      "topic0",
			Entries:    msgArr,
		})
		assert.Nil(t, err)

		pubsubIns := rt.pubSubs[testBulkSubscribePubsub].component.(*mockSubscribePubSub)
		assert.Equal(t, 1, pubsubIns.bulkPubCount["topic0"])
		assert.True(t, pubsubIns.isBulkSubscribe)
		reqs := mockAppChannel.GetInvokedRequest()
		mockAppChannel.AssertNumberOfCalls(t, "InvokeMethod", 2)
		assert.Contains(t, string(reqs["orders"].Message().Data.Value), "\"event\":"+order1)
		assert.Contains(t, string(reqs["orders"].Message().Data.Value), "\"event\":"+order2)
	})

	t.Run("bulk Subscribe events on different paths", func(t *testing.T) {
		rt := NewTestDaprRuntime(modes.StandaloneMode)
		defer stopRuntime(t, rt)
		rt.pubSubRegistry.RegisterComponent(
			func(_ logger.Logger) pubsub.PubSub {
				return &mockSubscribePubSub{}
			},
			"mockPubSub",
		)
		req := invokev1.NewInvokeMethodRequest("dapr/subscribe")
		req.WithHTTPExtension(http.MethodGet, "")
		req.WithRawData(nil, invokev1.JSONContentType)

		subscriptionItems := []runtimePubsub.SubscriptionJSON{
			{
				PubsubName: testBulkSubscribePubsub,
				Topic:      "topic0",
				Routes: runtimePubsub.RoutesJSON{
					Rules: []*runtimePubsub.RuleJSON{
						{
							Path:  "orders1",
							Match: `event.type == "type1"`,
						},
						{
							Path:  "orders2",
							Match: `event.type == "type2"`,
						},
					},
				},
				Metadata: map[string]string{"bulkSubscribe": "true"},
			},
		}
		sub, _ := json.Marshal(subscriptionItems)
		fakeResp := invokev1.NewInvokeMethodResponse(200, "OK", nil)
		fakeResp.WithRawData(sub, "application/json")

		mockAppChannel := new(channelt.MockAppChannel)
		mockAppChannel.Init()
		rt.appChannel = mockAppChannel
		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), req).Return(fakeResp, nil)
		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), mock.Anything).Return(fakeResp, nil)

		require.NoError(t, rt.initPubSub(pubsubComponent))
		rt.startSubscriptions()

		msgArr := getBulkMessageEntries(2)

		_, err := rt.BulkPublish(context.TODO(), &pubsub.BulkPublishRequest{
			PubsubName: testBulkSubscribePubsub,
			Topic:      "topic0",
			Entries:    msgArr,
		})
		assert.Nil(t, err)

		pubsubIns := rt.pubSubs[testBulkSubscribePubsub].component.(*mockSubscribePubSub)
		assert.Equal(t, 1, pubsubIns.bulkPubCount["topic0"])
		assert.True(t, pubsubIns.isBulkSubscribe)
		reqs := mockAppChannel.GetInvokedRequest()
		mockAppChannel.AssertNumberOfCalls(t, "InvokeMethod", 3)
		assert.Contains(t, string(reqs["orders1"].Message().Data.Value), "\"event\":"+order1)
		assert.NotContains(t, string(reqs["orders1"].Message().Data.Value), "\"event\":"+order2)
		assert.Contains(t, string(reqs["orders2"].Message().Data.Value), "\"event\":"+order2)
		assert.NotContains(t, string(reqs["orders2"].Message().Data.Value), "\"event\":"+order1)
	})

	t.Run("verify Responses when bulk Subscribe events on different paths", func(t *testing.T) {
		rt := NewTestDaprRuntime(modes.StandaloneMode)
		defer stopRuntime(t, rt)
		rt.pubSubRegistry.RegisterComponent(
			func(_ logger.Logger) pubsub.PubSub {
				return &mockSubscribePubSub{}
			},
			"mockPubSub",
		)
		req := invokev1.NewInvokeMethodRequest("dapr/subscribe")
		req.WithHTTPExtension(http.MethodGet, "")
		req.WithRawData(nil, invokev1.JSONContentType)

		subscriptionItems := []runtimePubsub.SubscriptionJSON{
			{
				PubsubName: testBulkSubscribePubsub,
				Topic:      "topic0",
				Routes: runtimePubsub.RoutesJSON{
					Rules: []*runtimePubsub.RuleJSON{
						{
							Path:  "orders1",
							Match: `event.type == "type1"`,
						},
						{
							Path:  "orders2",
							Match: `event.type == "type2"`,
						},
					},
				},
				Metadata: map[string]string{"bulkSubscribe": "true"},
			},
		}
		sub, _ := json.Marshal(subscriptionItems)
		fakeResp := invokev1.NewInvokeMethodResponse(200, "OK", nil)
		fakeResp.WithRawData(sub, "application/json")

		mockAppChannel := new(channelt.MockAppChannel)
		mockAppChannel.Init()
		rt.appChannel = mockAppChannel
		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), req).Return(fakeResp, nil)

		require.NoError(t, rt.initPubSub(pubsubComponent))
		rt.startSubscriptions()

		msgArr := getBulkMessageEntries(10)
		responseItemsOrders1 := pubsub.AppBulkResponse{
			AppResponses: []pubsub.AppBulkResponseEntry{
				{EntryID: "1111111a", Status: "SUCCESS"},
				{EntryID: "333333c", Status: "RETRY"},
				{EntryID: "5555555e", Status: "DROP"},
				{EntryID: "7777777g", Status: "RETRY"},
				{EntryID: "8888888h", Status: "SUCCESS"},
				{EntryID: "9999999i", Status: "SUCCESS"},
			},
		}

		resp1, _ := json.Marshal(responseItemsOrders1)
		respInvoke1 := invokev1.NewInvokeMethodResponse(200, "OK", nil)
		respInvoke1.WithRawData(resp1, "application/json")

		responseItemsOrders2 := pubsub.AppBulkResponse{
			AppResponses: []pubsub.AppBulkResponseEntry{
				{EntryID: "2222222b", Status: "SUCCESS"},
				{EntryID: "4444444d", Status: "DROP"},
				{EntryID: "66666666f", Status: "DROP"},
				{EntryID: "10101010j", Status: "SUCCESS"},
			},
		}

		resp2, _ := json.Marshal(responseItemsOrders2)
		respInvoke2 := invokev1.NewInvokeMethodResponse(200, "OK", nil)
		respInvoke2.WithRawData(resp2, "application/json")

		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), mock.MatchedBy(
			func(req *invokev1.InvokeMethodRequest) bool { return req.Message().Method == "orders1" })).Return(respInvoke1, nil)
		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), mock.MatchedBy(
			func(req *invokev1.InvokeMethodRequest) bool { return req.Message().Method == "orders2" })).Return(respInvoke2, nil)

		_, err := rt.BulkPublish(context.TODO(), &pubsub.BulkPublishRequest{
			PubsubName: testBulkSubscribePubsub,
			Topic:      "topic0",
			Entries:    msgArr,
		})
		assert.Nil(t, err)

		pubsubIns := rt.pubSubs[testBulkSubscribePubsub].component.(*mockSubscribePubSub)
		assert.Equal(t, 1, pubsubIns.bulkPubCount["topic0"])
		assert.True(t, pubsubIns.isBulkSubscribe)
		reqs := mockAppChannel.GetInvokedRequest()
		mockAppChannel.AssertNumberOfCalls(t, "InvokeMethod", 3)
		assert.True(t, verifyIfEventContainsStrings(reqs["orders1"].Message().Data.Value, "\"event\":"+order1,
			"\"event\":"+order3, "\"event\":"+order5, "\"event\":"+order7, "\"event\":"+order8, "\"event\":"+order9))
		assert.True(t, verifyIfEventNotContainsStrings(reqs["orders1"].Message().Data.Value, "\"event\":"+order2,
			"\"event\":"+order4, "\"event\":"+order6, "\"event\":"+order10))
		assert.True(t, verifyIfEventContainsStrings(reqs["orders2"].Message().Data.Value, "\"event\":"+order2,
			"\"event\":"+order4, "\"event\":"+order6, "\"event\":"+order10))
		assert.True(t, verifyIfEventNotContainsStrings(reqs["orders2"].Message().Data.Value, "\"event\":"+order1,
			"\"event\":"+order3, "\"event\":"+order5, "\"event\":"+order7, "\"event\":"+order8, "\"event\":"+order9))

		expectedResponse := BulkResponseExpectation{
			Responses: []BulkResponseEntryExpectation{
				{EntryID: "1111111a", IsError: false},
				{EntryID: "2222222b", IsError: false},
				{EntryID: "333333c", IsError: true},
				{EntryID: "4444444d", IsError: false},
				{EntryID: "5555555e", IsError: false},
				{EntryID: "66666666f", IsError: false},
				{EntryID: "7777777g", IsError: true},
				{EntryID: "8888888h", IsError: false},
				{EntryID: "9999999i", IsError: false},
				{EntryID: "10101010j", IsError: false},
			},
		}

		assert.True(t, verifyBulkSubscribeResponses(expectedResponse, pubsubIns.bulkReponse))
	})

	t.Run("verify Responses when entryID supplied blank while sending messages", func(t *testing.T) {
		rt := NewTestDaprRuntime(modes.StandaloneMode)
		defer stopRuntime(t, rt)
		rt.pubSubRegistry.RegisterComponent(
			func(_ logger.Logger) pubsub.PubSub {
				return &mockSubscribePubSub{}
			},
			"mockPubSub",
		)
		req := invokev1.NewInvokeMethodRequest("dapr/subscribe")
		req.WithHTTPExtension(http.MethodGet, "")
		req.WithRawData(nil, invokev1.JSONContentType)

		subscriptionItems := []runtimePubsub.SubscriptionJSON{
			{
				PubsubName: testBulkSubscribePubsub,
				Topic:      "topic0",
				Route:      "orders",
				Metadata:   map[string]string{"bulkSubscribe": "true"},
			},
		}
		sub, _ := json.Marshal(subscriptionItems)
		fakeResp := invokev1.NewInvokeMethodResponse(200, "OK", nil)
		fakeResp.WithRawData(sub, "application/json")

		mockAppChannel := new(channelt.MockAppChannel)
		mockAppChannel.Init()
		rt.appChannel = mockAppChannel
		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), req).Return(fakeResp, nil)

		require.NoError(t, rt.initPubSub(pubsubComponent))
		rt.startSubscriptions()

		msgArr := getBulkMessageEntries(4)
		msgArr[0].EntryID = ""
		msgArr[2].EntryID = ""

		responseItemsOrders1 := pubsub.AppBulkResponse{
			AppResponses: []pubsub.AppBulkResponseEntry{
				{EntryID: "2222222b", Status: "SUCCESS"},
				{EntryID: "4444444d", Status: "SUCCESS"},
			},
		}

		resp1, _ := json.Marshal(responseItemsOrders1)
		respInvoke1 := invokev1.NewInvokeMethodResponse(200, "OK", nil)
		respInvoke1.WithRawData(resp1, "application/json")

		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), mock.MatchedBy(
			func(req *invokev1.InvokeMethodRequest) bool { return req.Message().Method == "orders" })).Return(respInvoke1, nil)

		_, err := rt.BulkPublish(context.TODO(), &pubsub.BulkPublishRequest{
			PubsubName: testBulkSubscribePubsub,
			Topic:      "topic0",
			Entries:    msgArr,
		})
		assert.Nil(t, err)

		pubsubIns := rt.pubSubs[testBulkSubscribePubsub].component.(*mockSubscribePubSub)
		assert.Equal(t, 1, pubsubIns.bulkPubCount["topic0"])
		assert.True(t, pubsubIns.isBulkSubscribe)
		reqs := mockAppChannel.GetInvokedRequest()
		mockAppChannel.AssertNumberOfCalls(t, "InvokeMethod", 2)
		assert.True(t, verifyIfEventContainsStrings(reqs["orders"].Message().Data.Value, "\"event\":"+order2,
			"\"event\":"+order4))
		assert.True(t, verifyIfEventNotContainsStrings(reqs["orders"].Message().Data.Value, "\"event\":"+order1,
			"\"event\":"+order3))

		expectedResponse := BulkResponseExpectation{
			Responses: []BulkResponseEntryExpectation{
				{EntryID: "", IsError: true},
				{EntryID: "2222222b", IsError: false},
				{EntryID: "", IsError: true},
				{EntryID: "4444444d", IsError: false},
			},
		}

		assert.True(t, verifyBulkSubscribeResponses(expectedResponse, pubsubIns.bulkReponse))
	})

	t.Run("verify bulk Subscribe Responses when App sends back out of order entryIDs", func(t *testing.T) {
		rt := NewTestDaprRuntime(modes.StandaloneMode)
		defer stopRuntime(t, rt)
		rt.pubSubRegistry.RegisterComponent(
			func(_ logger.Logger) pubsub.PubSub {
				return &mockSubscribePubSub{}
			},
			"mockPubSub",
		)
		req := invokev1.NewInvokeMethodRequest("dapr/subscribe")
		req.WithHTTPExtension(http.MethodGet, "")
		req.WithRawData(nil, invokev1.JSONContentType)

		subscriptionItems := []runtimePubsub.SubscriptionJSON{
			{
				PubsubName: testBulkSubscribePubsub,
				Topic:      "topic0",
				Route:      "orders",
				Metadata:   map[string]string{"bulkSubscribe": "true"},
			},
		}
		sub, _ := json.Marshal(subscriptionItems)
		fakeResp := invokev1.NewInvokeMethodResponse(200, "OK", nil)
		fakeResp.WithRawData(sub, "application/json")

		mockAppChannel := new(channelt.MockAppChannel)
		mockAppChannel.Init()
		rt.appChannel = mockAppChannel
		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), req).Return(fakeResp, nil)

		require.NoError(t, rt.initPubSub(pubsubComponent))
		rt.startSubscriptions()

		msgArr := getBulkMessageEntries(5)

		responseItemsOrders1 := pubsub.AppBulkResponse{
			AppResponses: []pubsub.AppBulkResponseEntry{
				{EntryID: "2222222b", Status: "RETRY"},
				{EntryID: "333333c", Status: "SUCCESS"},
				{EntryID: "5555555e", Status: "RETRY"},
				{EntryID: "1111111a", Status: "SUCCESS"},
				{EntryID: "4444444d", Status: "SUCCESS"},
			},
		}

		resp1, _ := json.Marshal(responseItemsOrders1)
		respInvoke1 := invokev1.NewInvokeMethodResponse(200, "OK", nil)
		respInvoke1.WithRawData(resp1, "application/json")

		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), mock.MatchedBy(
			func(req *invokev1.InvokeMethodRequest) bool { return req.Message().Method == "orders" })).Return(respInvoke1, nil)

		_, err := rt.BulkPublish(context.TODO(), &pubsub.BulkPublishRequest{
			PubsubName: testBulkSubscribePubsub,
			Topic:      "topic0",
			Entries:    msgArr,
		})
		assert.Nil(t, err)

		pubsubIns := rt.pubSubs[testBulkSubscribePubsub].component.(*mockSubscribePubSub)
		assert.Equal(t, 1, pubsubIns.bulkPubCount["topic0"])
		assert.True(t, pubsubIns.isBulkSubscribe)
		reqs := mockAppChannel.GetInvokedRequest()
		mockAppChannel.AssertNumberOfCalls(t, "InvokeMethod", 2)
		assert.True(t, verifyIfEventContainsStrings(reqs["orders"].Message().Data.Value, "\"event\":"+order1,
			"\"event\":"+order2, "\"event\":"+order3, "\"event\":"+order4, "\"event\":"+order5))

		expectedResponse := BulkResponseExpectation{
			Responses: []BulkResponseEntryExpectation{
				{EntryID: "1111111a", IsError: false},
				{EntryID: "2222222b", IsError: true},
				{EntryID: "333333c", IsError: false},
				{EntryID: "4444444d", IsError: false},
				{EntryID: "5555555e", IsError: true},
			},
		}

		assert.True(t, verifyBulkSubscribeResponses(expectedResponse, pubsubIns.bulkReponse))
	})

	t.Run("verify bulk Subscribe Responses when App sends back wrong entryIDs", func(t *testing.T) {
		rt := NewTestDaprRuntime(modes.StandaloneMode)
		defer stopRuntime(t, rt)
		rt.pubSubRegistry.RegisterComponent(
			func(_ logger.Logger) pubsub.PubSub {
				return &mockSubscribePubSub{}
			},
			"mockPubSub",
		)
		req := invokev1.NewInvokeMethodRequest("dapr/subscribe")
		req.WithHTTPExtension(http.MethodGet, "")
		req.WithRawData(nil, invokev1.JSONContentType)

		subscriptionItems := []runtimePubsub.SubscriptionJSON{
			{
				PubsubName: testBulkSubscribePubsub,
				Topic:      "topic0",
				Route:      "orders",
				Metadata:   map[string]string{"bulkSubscribe": "true"},
			},
		}
		sub, _ := json.Marshal(subscriptionItems)
		fakeResp := invokev1.NewInvokeMethodResponse(200, "OK", nil)
		fakeResp.WithRawData(sub, "application/json")

		mockAppChannel := new(channelt.MockAppChannel)
		mockAppChannel.Init()
		rt.appChannel = mockAppChannel
		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), req).Return(fakeResp, nil)

		require.NoError(t, rt.initPubSub(pubsubComponent))
		rt.startSubscriptions()

		msgArr := getBulkMessageEntries(5)

		responseItemsOrders1 := pubsub.AppBulkResponse{
			AppResponses: []pubsub.AppBulkResponseEntry{
				{EntryID: "wrongEntryID1", Status: "SUCCESS"},
				{EntryID: "2222222b", Status: "RETRY"},
				{EntryID: "333333c", Status: "SUCCESS"},
				{EntryID: "wrongEntryID2", Status: "SUCCESS"},
				{EntryID: "5555555e", Status: "RETRY"},
			},
		}

		resp1, _ := json.Marshal(responseItemsOrders1)
		respInvoke1 := invokev1.NewInvokeMethodResponse(200, "OK", nil)
		respInvoke1.WithRawData(resp1, "application/json")

		mockAppChannel.On("InvokeMethod", mock.MatchedBy(matchContextInterface), mock.MatchedBy(
			func(req *invokev1.InvokeMethodRequest) bool { return req.Message().Method == "orders" })).Return(respInvoke1, nil)

		_, err := rt.BulkPublish(context.TODO(), &pubsub.BulkPublishRequest{
			PubsubName: testBulkSubscribePubsub,
			Topic:      "topic0",
			Entries:    msgArr,
		})
		assert.Nil(t, err)

		pubsubIns := rt.pubSubs[testBulkSubscribePubsub].component.(*mockSubscribePubSub)
		assert.Equal(t, 1, pubsubIns.bulkPubCount["topic0"])
		assert.True(t, pubsubIns.isBulkSubscribe)
		reqs := mockAppChannel.GetInvokedRequest()
		mockAppChannel.AssertNumberOfCalls(t, "InvokeMethod", 2)
		assert.True(t, verifyIfEventContainsStrings(reqs["orders"].Message().Data.Value, "\"event\":"+order1,
			"\"event\":"+order2, "\"event\":"+order3, "\"event\":"+order4, "\"event\":"+order5))

		expectedResponse := BulkResponseExpectation{
			Responses: []BulkResponseEntryExpectation{
				{EntryID: "1111111a", IsError: true},
				{EntryID: "2222222b", IsError: true},
				{EntryID: "333333c", IsError: false},
				{EntryID: "4444444d", IsError: true},
				{EntryID: "5555555e", IsError: true},
			},
		}

		assert.True(t, verifyBulkSubscribeResponses(expectedResponse, pubsubIns.bulkReponse))
	})
}

func TestBulkSubscribeGRPC(t *testing.T) {

	testBulkSubscribePubsub := "bulkSubscribePubSub"
	pubsubComponent := componentsV1alpha1.Component{
		ObjectMeta: metaV1.ObjectMeta{
			Name: testBulkSubscribePubsub,
		},
		Spec: componentsV1alpha1.ComponentSpec{
			Type:     "pubsub.mockPubSub",
			Version:  "v1",
			Metadata: getFakeMetadataItems(),
		},
	}

	t.Run("GRPC - bulk Subscribe Message for raw payload", func(t *testing.T) {
		port, _ := freeport.GetFreePort()
		rt := NewTestDaprRuntimeWithProtocol(modes.StandaloneMode, string(GRPCProtocol), port)
		defer stopRuntime(t, rt)

		rt.pubSubRegistry.RegisterComponent(
			func(_ logger.Logger) pubsub.PubSub {
				return &mockSubscribePubSub{}
			},
			"mockPubSub",
		)

		subscriptionItems := runtimev1pb.ListTopicSubscriptionsResponse{
			Subscriptions: []*commonV1.TopicSubscription{
				{
					PubsubName: testBulkSubscribePubsub,
					Topic:      "topic0",
					Routes: &commonV1.TopicRoutes{
						Default: "orders",
					},
					Metadata: map[string]string{"bulkSubscribe": "true", "rawPayload": "true"},
				},
			},
		}

		nbei1 := pubsub.BulkMessageEntry{EntryID: "1111111a", Event: []byte(`{"orderId":"1"}`)}
		nbei2 := pubsub.BulkMessageEntry{EntryID: "2222222b", Event: []byte(`{"orderId":"2"}`)}
		msgArr := []pubsub.BulkMessageEntry{nbei1, nbei2}
		responseEntries := make([]*runtimev1pb.TopicEventBulkResponseEntry, 2)
		for k, msg := range msgArr {
			responseEntries[k] = &runtimev1pb.TopicEventBulkResponseEntry{
				EntryID: msg.EntryID,
			}
		}
		responseEntries = setBulkResponseStatus(responseEntries,
			runtimev1pb.TopicEventResponse_DROP,
			runtimev1pb.TopicEventResponse_SUCCESS)

		responses := runtimev1pb.TopicEventBulkResponse{
			Statuses: responseEntries,
		}
		mapResp := make(map[string]*runtimev1pb.TopicEventBulkResponse)
		mapResp["orders"] = &responses
		// create mock application server first
		mockServer := &channelt.MockServer{
			ListTopicSubscriptionsResponse: subscriptionItems,
			BulkResponsePerPath:            mapResp,
			Error:                          nil,
		}
		grpcServer := startTestAppCallbackGRPCServer(t, port, mockServer)
		if grpcServer != nil {
			// properly stop the gRPC server
			defer grpcServer.Stop()
		}

		// create a new AppChannel and gRPC client for every test
		rt.createAppChannel()
		// properly close the app channel created
		defer rt.grpc.AppClient.Close()

		require.NoError(t, rt.initPubSub(pubsubComponent))
		rt.startSubscriptions()

		_, err := rt.BulkPublish(context.TODO(), &pubsub.BulkPublishRequest{
			PubsubName: testBulkSubscribePubsub,
			Topic:      "topic0",
			Entries:    msgArr,
		})
		assert.Nil(t, err)
		pubsubIns := rt.pubSubs[testBulkSubscribePubsub].component.(*mockSubscribePubSub)

		expectedResponse := BulkResponseExpectation{
			Responses: []BulkResponseEntryExpectation{
				{EntryID: "1111111a", IsError: false},
				{EntryID: "2222222b", IsError: false},
			},
		}
		assert.Contains(t, string((mockServer.RequestsReceived["orders"].GetEntries()[0].GetEvent())), `{"orderId":"1"}`)
		assert.Contains(t, string((mockServer.RequestsReceived["orders"].GetEntries()[1].GetEvent())), `{"orderId":"2"}`)
		assert.True(t, verifyBulkSubscribeResponses(expectedResponse, pubsubIns.bulkReponse))
	})

	t.Run("GRPC - bulk Subscribe cloud event Message on different paths and verify response", func(t *testing.T) {
		port, _ := freeport.GetFreePort()
		rt := NewTestDaprRuntimeWithProtocol(modes.StandaloneMode, string(GRPCProtocol), port)
		defer stopRuntime(t, rt)

		rt.pubSubRegistry.RegisterComponent(
			func(_ logger.Logger) pubsub.PubSub {
				return &mockSubscribePubSub{}
			},
			"mockPubSub",
		)

		subscriptionItems := runtimev1pb.ListTopicSubscriptionsResponse{
			Subscriptions: []*commonV1.TopicSubscription{
				{
					PubsubName: testBulkSubscribePubsub,
					Topic:      "topic0",
					Routes: &commonV1.TopicRoutes{
						Rules: []*commonV1.TopicRule{
							{
								Path:  "orders1",
								Match: `event.type == "type1"`,
							},
							{
								Path:  "orders2",
								Match: `event.type == "type2"`,
							},
						},
					},
					Metadata: map[string]string{"bulkSubscribe": "true"},
				},
			},
		}
		msgArr := getBulkMessageEntries(10)
		responseEntries1 := make([]*runtimev1pb.TopicEventBulkResponseEntry, 6)
		responseEntries2 := make([]*runtimev1pb.TopicEventBulkResponseEntry, 4)
		i := 0
		j := 0
		for k, msg := range msgArr {
			if strings.Contains(string(msgArr[k].Event), "type1") {
				responseEntries1[i] = &runtimev1pb.TopicEventBulkResponseEntry{
					EntryID: msg.EntryID,
				}
				i++
			} else if strings.Contains(string(msgArr[k].Event), "type2") {
				responseEntries2[j] = &runtimev1pb.TopicEventBulkResponseEntry{
					EntryID: msg.EntryID,
				}
				j++
			}
		}
		responseEntries1 = setBulkResponseStatus(responseEntries1,
			runtimev1pb.TopicEventResponse_DROP,
			runtimev1pb.TopicEventResponse_RETRY,
			runtimev1pb.TopicEventResponse_DROP,
			runtimev1pb.TopicEventResponse_RETRY,
			runtimev1pb.TopicEventResponse_SUCCESS,
			runtimev1pb.TopicEventResponse_DROP)

		responseEntries2 = setBulkResponseStatus(responseEntries2,
			runtimev1pb.TopicEventResponse_RETRY,
			runtimev1pb.TopicEventResponse_DROP,
			runtimev1pb.TopicEventResponse_RETRY,
			runtimev1pb.TopicEventResponse_SUCCESS)

		responses1 := runtimev1pb.TopicEventBulkResponse{
			Statuses: responseEntries1,
		}
		responses2 := runtimev1pb.TopicEventBulkResponse{
			Statuses: responseEntries2,
		}
		mapResp := make(map[string]*runtimev1pb.TopicEventBulkResponse)
		mapResp["orders1"] = &responses1
		mapResp["orders2"] = &responses2
		// create mock application server first
		grpcServer := startTestAppCallbackGRPCServer(t, port, &channelt.MockServer{
			ListTopicSubscriptionsResponse: subscriptionItems,
			BulkResponsePerPath:            mapResp,
			Error:                          nil,
		})
		if grpcServer != nil {
			defer grpcServer.Stop()
		}

		rt.createAppChannel()
		defer rt.grpc.AppClient.Close()

		require.NoError(t, rt.initPubSub(pubsubComponent))
		rt.startSubscriptions()

		_, err := rt.BulkPublish(context.TODO(), &pubsub.BulkPublishRequest{
			PubsubName: testBulkSubscribePubsub,
			Topic:      "topic0",
			Entries:    msgArr,
		})
		assert.Nil(t, err)
		pubsubIns := rt.pubSubs[testBulkSubscribePubsub].component.(*mockSubscribePubSub)

		expectedResponse := BulkResponseExpectation{
			Responses: []BulkResponseEntryExpectation{
				{EntryID: "1111111a", IsError: false},
				{EntryID: "2222222b", IsError: true},
				{EntryID: "333333c", IsError: true},
				{EntryID: "4444444d", IsError: false},
				{EntryID: "5555555e", IsError: false},
				{EntryID: "66666666f", IsError: true},
				{EntryID: "7777777g", IsError: true},
				{EntryID: "8888888h", IsError: false},
				{EntryID: "9999999i", IsError: false},
				{EntryID: "10101010j", IsError: false},
			},
		}

		assert.True(t, verifyBulkSubscribeResponses(expectedResponse, pubsubIns.bulkReponse))
	})

	t.Run("GRPC - verify Responses when entryID supplied blank while sending messages", func(t *testing.T) {
		port, _ := freeport.GetFreePort()
		rt := NewTestDaprRuntimeWithProtocol(modes.StandaloneMode, string(GRPCProtocol), port)
		defer stopRuntime(t, rt)

		rt.pubSubRegistry.RegisterComponent(
			func(_ logger.Logger) pubsub.PubSub {
				return &mockSubscribePubSub{}
			},
			"mockPubSub",
		)

		subscriptionItems := runtimev1pb.ListTopicSubscriptionsResponse{
			Subscriptions: []*commonV1.TopicSubscription{
				{
					PubsubName: testBulkSubscribePubsub,
					Topic:      "topic0",
					Routes: &commonV1.TopicRoutes{
						Default: "orders",
					},
					Metadata: map[string]string{"bulkSubscribe": "true"},
				},
			},
		}
		msgArr := getBulkMessageEntries(4)
		msgArr[0].EntryID = ""
		msgArr[2].EntryID = ""
		responseEntries := make([]*runtimev1pb.TopicEventBulkResponseEntry, 4)
		for k, msg := range msgArr {
			responseEntries[k] = &runtimev1pb.TopicEventBulkResponseEntry{
				EntryID: msg.EntryID,
			}
		}
		responseEntries[1].Status = runtimev1pb.TopicEventResponse_SUCCESS
		responseEntries[3].Status = runtimev1pb.TopicEventResponse_SUCCESS
		responses := runtimev1pb.TopicEventBulkResponse{
			Statuses: responseEntries,
		}
		mapResp := make(map[string]*runtimev1pb.TopicEventBulkResponse)
		mapResp["orders"] = &responses
		// create mock application server first
		grpcServer := startTestAppCallbackGRPCServer(t, port, &channelt.MockServer{
			ListTopicSubscriptionsResponse: subscriptionItems,
			BulkResponsePerPath:            mapResp,
			Error:                          nil,
		})
		if grpcServer != nil {
			defer grpcServer.Stop()
		}

		rt.createAppChannel()
		defer rt.grpc.AppClient.Close()

		require.NoError(t, rt.initPubSub(pubsubComponent))
		rt.startSubscriptions()

		_, err := rt.BulkPublish(context.TODO(), &pubsub.BulkPublishRequest{
			PubsubName: testBulkSubscribePubsub,
			Topic:      "topic0",
			Entries:    msgArr,
		})
		assert.Nil(t, err)
		pubsubIns := rt.pubSubs[testBulkSubscribePubsub].component.(*mockSubscribePubSub)

		expectedResponse := BulkResponseExpectation{
			Responses: []BulkResponseEntryExpectation{
				{EntryID: "", IsError: true},
				{EntryID: "2222222b", IsError: false},
				{EntryID: "", IsError: true},
				{EntryID: "4444444d", IsError: false},
			},
		}

		assert.True(t, verifyBulkSubscribeResponses(expectedResponse, pubsubIns.bulkReponse))

	})

	t.Run("GRPC - verify bulk Subscribe Responses when App sends back out of order entryIDs", func(t *testing.T) {
		port, _ := freeport.GetFreePort()
		rt := NewTestDaprRuntimeWithProtocol(modes.StandaloneMode, string(GRPCProtocol), port)
		defer stopRuntime(t, rt)

		rt.pubSubRegistry.RegisterComponent(
			func(_ logger.Logger) pubsub.PubSub {
				return &mockSubscribePubSub{}
			},
			"mockPubSub",
		)

		subscriptionItems := runtimev1pb.ListTopicSubscriptionsResponse{
			Subscriptions: []*commonV1.TopicSubscription{
				{
					PubsubName: testBulkSubscribePubsub,
					Topic:      "topic0",
					Routes: &commonV1.TopicRoutes{
						Default: "orders",
					},
					Metadata: map[string]string{"bulkSubscribe": "true"},
				},
			},
		}
		msgArr := getBulkMessageEntries(5)
		responseEntries := make([]*runtimev1pb.TopicEventBulkResponseEntry, 5)
		responseEntries[0] = &runtimev1pb.TopicEventBulkResponseEntry{
			EntryID: msgArr[1].EntryID,
			Status:  runtimev1pb.TopicEventResponse_RETRY,
		}
		responseEntries[1] = &runtimev1pb.TopicEventBulkResponseEntry{
			EntryID: msgArr[2].EntryID,
			Status:  runtimev1pb.TopicEventResponse_SUCCESS,
		}
		responseEntries[2] = &runtimev1pb.TopicEventBulkResponseEntry{
			EntryID: msgArr[4].EntryID,
			Status:  runtimev1pb.TopicEventResponse_RETRY,
		}
		responseEntries[3] = &runtimev1pb.TopicEventBulkResponseEntry{
			EntryID: msgArr[0].EntryID,
			Status:  runtimev1pb.TopicEventResponse_SUCCESS,
		}
		responseEntries[4] = &runtimev1pb.TopicEventBulkResponseEntry{
			EntryID: msgArr[3].EntryID,
			Status:  runtimev1pb.TopicEventResponse_SUCCESS,
		}
		responses := runtimev1pb.TopicEventBulkResponse{
			Statuses: responseEntries,
		}
		mapResp := make(map[string]*runtimev1pb.TopicEventBulkResponse)
		mapResp["orders"] = &responses
		// create mock application server first
		grpcServer := startTestAppCallbackGRPCServer(t, port, &channelt.MockServer{
			ListTopicSubscriptionsResponse: subscriptionItems,
			BulkResponsePerPath:            mapResp,
			Error:                          nil,
		})
		if grpcServer != nil {
			defer grpcServer.Stop()
		}

		rt.createAppChannel()
		defer rt.grpc.AppClient.Close()

		require.NoError(t, rt.initPubSub(pubsubComponent))
		rt.startSubscriptions()

		_, err := rt.BulkPublish(context.TODO(), &pubsub.BulkPublishRequest{
			PubsubName: testBulkSubscribePubsub,
			Topic:      "topic0",
			Entries:    msgArr,
		})
		assert.Nil(t, err)
		pubsubIns := rt.pubSubs[testBulkSubscribePubsub].component.(*mockSubscribePubSub)

		expectedResponse := BulkResponseExpectation{
			Responses: []BulkResponseEntryExpectation{
				{EntryID: "1111111a", IsError: false},
				{EntryID: "2222222b", IsError: true},
				{EntryID: "333333c", IsError: false},
				{EntryID: "4444444d", IsError: false},
				{EntryID: "5555555e", IsError: true},
			},
		}

		assert.True(t, verifyBulkSubscribeResponses(expectedResponse, pubsubIns.bulkReponse))
	})

	t.Run("GRPC - verify bulk Subscribe Responses when App sends back wrong entryIDs", func(t *testing.T) {
		port, _ := freeport.GetFreePort()
		rt := NewTestDaprRuntimeWithProtocol(modes.StandaloneMode, string(GRPCProtocol), port)
		defer stopRuntime(t, rt)

		rt.pubSubRegistry.RegisterComponent(
			func(_ logger.Logger) pubsub.PubSub {
				return &mockSubscribePubSub{}
			},
			"mockPubSub",
		)

		subscriptionItems := runtimev1pb.ListTopicSubscriptionsResponse{
			Subscriptions: []*commonV1.TopicSubscription{
				{
					PubsubName: testBulkSubscribePubsub,
					Topic:      "topic0",
					Routes: &commonV1.TopicRoutes{
						Default: "orders",
					},
					Metadata: map[string]string{"bulkSubscribe": "true"},
				},
			},
		}
		msgArr := getBulkMessageEntries(5)
		responseEntries := make([]*runtimev1pb.TopicEventBulkResponseEntry, 5)
		for k, msg := range msgArr {
			responseEntries[k] = &runtimev1pb.TopicEventBulkResponseEntry{
				EntryID: msg.EntryID,
			}
		}
		responseEntries[0].EntryID = "wrongId1"
		responseEntries[3].EntryID = "wrongId2"
		responseEntries = setBulkResponseStatus(responseEntries,
			runtimev1pb.TopicEventResponse_SUCCESS,
			runtimev1pb.TopicEventResponse_RETRY,
			runtimev1pb.TopicEventResponse_SUCCESS,
			runtimev1pb.TopicEventResponse_SUCCESS,
			runtimev1pb.TopicEventResponse_RETRY)

		responses := runtimev1pb.TopicEventBulkResponse{
			Statuses: responseEntries,
		}
		mapResp := make(map[string]*runtimev1pb.TopicEventBulkResponse)
		mapResp["orders"] = &responses
		// create mock application server first
		grpcServer := startTestAppCallbackGRPCServer(t, port, &channelt.MockServer{
			ListTopicSubscriptionsResponse: subscriptionItems,
			BulkResponsePerPath:            mapResp,
			Error:                          nil,
		})
		if grpcServer != nil {
			defer grpcServer.Stop()
		}

		rt.createAppChannel()
		defer rt.grpc.AppClient.Close()

		require.NoError(t, rt.initPubSub(pubsubComponent))
		rt.startSubscriptions()

		_, err := rt.BulkPublish(context.TODO(), &pubsub.BulkPublishRequest{
			PubsubName: testBulkSubscribePubsub,
			Topic:      "topic0",
			Entries:    msgArr,
		})
		assert.Nil(t, err)
		pubsubIns := rt.pubSubs[testBulkSubscribePubsub].component.(*mockSubscribePubSub)

		expectedResponse := BulkResponseExpectation{
			Responses: []BulkResponseEntryExpectation{
				{EntryID: "1111111a", IsError: true},
				{EntryID: "2222222b", IsError: true},
				{EntryID: "333333c", IsError: false},
				{EntryID: "4444444d", IsError: true},
				{EntryID: "5555555e", IsError: true},
			},
		}

		assert.True(t, verifyBulkSubscribeResponses(expectedResponse, pubsubIns.bulkReponse))
	})
}

func setBulkResponseStatus(responses []*runtimev1pb.TopicEventBulkResponseEntry,
	status ...runtimev1pb.TopicEventResponse_TopicEventResponseStatus,
) []*runtimev1pb.TopicEventBulkResponseEntry {
	for i, s := range status {
		responses[i].Status = s
	}
	return responses
}

type BulkResponseEntryExpectation struct {
	EntryID string
	IsError bool
}

type BulkResponseExpectation struct {
	Responses []BulkResponseEntryExpectation
}

func verifyBulkSubscribeResponses(expected BulkResponseExpectation, actual pubsub.BulkSubscribeResponse) bool {
	for i, expectedEntryResponse := range expected.Responses {
		if expectedEntryResponse.EntryID != actual.Statuses[i].EntryID {
			return false
		}
		if (actual.Statuses[i].Error != nil) != expectedEntryResponse.IsError {
			return false
		}
	}
	return true
}

func verifyIfEventContainsStrings(event []byte, elems ...string) bool {
	for _, elem := range elems {
		if !strings.Contains(string(event), elem) {
			return false
		}
	}
	return true
}

func verifyIfEventNotContainsStrings(event []byte, elems ...string) bool {
	for _, elem := range elems {
		if strings.Contains(string(event), elem) {
			return false
		}
	}
	return true
}
