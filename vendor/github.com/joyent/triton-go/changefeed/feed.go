package changefeed

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"time"

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

func (c *FeedClient) Subscribe(ctx context.Context) <-chan *ChangeItem {
	fullPath := path.Join("/", c.client.AccountName, "changefeed")
	reqInputs := client.RequestInput{
		Method: http.MethodGet,
		Path:   fullPath,
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	conn, err := c.client.ExecuteRequestChangeFeed(ctx, reqInputs)
	if err != nil {
		fmt.Println(conn, err)
	}
	defer conn.Close()

	var register = &ChangeKind{
		Resource: "vm",
		SubResources: []string{
			"state",
		},
	}

	msg, _ := json.Marshal(register)

	changeChannel := make(chan *ChangeItem)
	done := make(chan struct{})

	// This Goroutine is our read/write loop. It keeps going until it cannot use the WebSocket anymore.
	go func() {
		defer conn.Close()
		defer close(done)

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
			var change *ChangeItem
			err = json.Unmarshal(message, &change)
			if err != nil {
				fmt.Println(err)
			}

			changeChannel <- change
			fmt.Printf("%+v\n", change)
		}
	}()

	for {
		select {
		case <-changeChannel:
			log.Println("Got ChangeItem.")
			return changeChannel
		// TODO: this is handy for testing, but maybe not needed in the triton-go lib, transfer to example?
		// Block until interrupted. Then send the close message to the server and wait for our other read/write Goroutine
		// to signal 'done', to safely terminate the WebSocket connection.
		case <-interrupt:
			log.Println("Client interrupted.")
			err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("WebSocket Close Error: ", err)
			}
			// Wait for 'done' or one second to pass.
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return nil
		// WebSocket has terminated before interrupt.
		case <-done:
			log.Println("WebSocket connection terminated.")
			return nil
		}
	}
}
