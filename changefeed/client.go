//
// Copyright (c) 2019, Joyent, Inc. All rights reserved.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//

package changefeed

import (
	"net/http"

	triton "github.com/joyent/triton-go"
	"github.com/joyent/triton-go/client"
)

type ChangeFeedClient struct {
	Client *client.Client
}

func newChangeFeedClient(client *client.Client) *ChangeFeedClient {
	return &ChangeFeedClient{
		Client: client,
	}
}

// NewClient returns a new client for working with ChangeFeed endpoints and
// resources within CloudAPI
func NewClient(config *triton.ClientConfig) (*ChangeFeedClient, error) {
	// TODO: Utilize config interface within the function itself
	client, err := client.New(
		config.TritonURL,
		config.MantaURL,
		config.AccountName,
		config.Signers...,
	)
	if err != nil {
		return nil, err
	}
	return newChangeFeedClient(client), nil
}

// SetHeaders allows a consumer of the current client to set custom headers for
// the next backend HTTP request sent to CloudAPI
func (c *ChangeFeedClient) SetHeader(header *http.Header) {
	c.Client.RequestHeader = header
}

// Machine returns a Compute client used for accessing functions pertaining to
// machine functionality in the Triton API.
func (c *ChangeFeedClient) Feed() *FeedClient {
	return &FeedClient{c.Client}
}
