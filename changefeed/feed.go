package changefeed

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path"

	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/joyent/triton-go/client"
	"github.com/joyent/triton-go/compute"
)

// VMAPI Resources
//"resource": "vm",
//"subResources": [
//"alias",
//"customer_metadata",
//"destroyed",
//"internal_metadata",
//"nics",
//"owner_uuid",
//"server_uuid",
//"state",
//"tags"
//],

// NAPI Resources

type FeedClient struct {
	client *client.Client
}

type ChangeItem struct {
	ChangeKind       ChangeKind       `json:"changeKind"`
	ChangeResourceID string           `json:"changedResourceId"`
	ResourceObject   compute.Instance `json:"resourceObject"`
	Published        string           `json:"published"`
}

type ChangeKind struct {
	Resource     string   `json:"resource"`
	SubResources []string `json:"subResources"`
}

type SubscribeOptions struct {
	Instances []string `json:"vms"`
}

func (c *FeedClient) Subscribe(ctx context.Context, opts *SubscribeOptions) <-chan *ChangeItem {
	fullPath := path.Join("/", c.client.AccountName, "changefeed")
	reqInputs := client.RequestInput{
		Method: http.MethodGet,
		Path:   fullPath,
	}

	conn, err := c.client.ExecuteRequestChangeFeed(ctx, reqInputs)
	if err != nil {
		fmt.Println(conn, err)
	}
	defer conn.Close()

	var register = &ChangeKind{
		Resource: "vm",
		SubResources: []string{
			"alias",
			"customer_metadata",
			"destroyed",
			"nics",
			"owner_uuid",
			"server_uuid",
			"state",
			"tags",
		},
	}

	msg, _ := json.Marshal(register)

	change := make(chan *ChangeItem)

	// This Goroutine is our read/write loop. It keeps going until it cannot use the WebSocket anymore.
	go func() {
		defer conn.Close()

		// Send Registration Message
		err = conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			fmt.Println(err)
		}

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("error: %v", err)
				}
				break
			}
			var ci *ChangeItem
			err = json.Unmarshal(message, &ci)
			if err != nil {
				fmt.Println(err)
			}
			change <- ci

			fmt.Printf("%+v\n", change)
		}
	}()

	// block and wait for cancel
	select {
	case <-ctx.Done():
	}
}
