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

type ChangeEvent struct {
	ChangeItem *ChangeItem       `json:"changeItem"`
	Instance   *compute.Instance `json:"instance"`
}

type ChangeItem struct {
	ChangeKind       *ChangeKind `json:"changeKind"`
	ChangeResourceID string      `json:"changedResourceId"`
	Published        string      `json:"published"`
}

type ChangeKind struct {
	Resource     string   `json:"resource"`
	SubResources []string `json:"subResources"`
}

func (c *FeedClient) Get(ctx context.Context) {
	fullPath := path.Join("/", c.client.AccountName, "changefeed")
	reqInputs := client.RequestInput{
		Method: http.MethodGet,
		Path:   fullPath,
	}
	conn, err := c.client.ExecuteRequestChangeFeed(ctx, reqInputs)
	if err != nil {
		fmt.Println(conn, err)
	}

	defer func() {
		conn.Close()
	}()

	var register = &ChangeKind{
		Resource: "vm",
		SubResources: []string{
			"state",
		},
	}

	msg, _ := json.Marshal(register)

	// Send Registration Message
	err = conn.WriteMessage(websocket.TextMessage, msg)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		var change *ChangeEvent
		err = json.Unmarshal(message, &change)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("%+v\n", change)
	}

	return
}
