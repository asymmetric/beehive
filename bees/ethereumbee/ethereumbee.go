/*
 *    Copyright (C) 2018 Lorenzo Manacorda
 *
 *    This program is free software: you can redistribute it and/or modify
 *    it under the terms of the GNU Affero General Public License as published
 *    by the Free Software Foundation, either version 3 of the License, or
 *    (at your option) any later version.
 *
 *    This program is distributed in the hope that it will be useful,
 *    but WITHOUT ANY WARRANTY; without even the implied warranty of
 *    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *    GNU Affero General Public License for more details.
 *
 *    You should have received a copy of the GNU Affero General Public License
 *    along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 *    Authors:
 *      Lorenzo Manacorda <lorenzo@mailbox.org>
 */

// Package ethereumbee is a Bee that can interface with the Ethereum blockchain.

package ethereumbee

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/muesli/beehive/bees"
)

type EthereumBee struct {
	bees.Bee
}

func (mod *EthereumBee) Run(eventChan chan bees.Event) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	url := mod.Options().Value("url").(string)
	mod.Logf("connecting to %s\n", url)

	// Timeout if the connection hasn't been established after 10 seconds.
	clientCtx, clientCancel := context.WithTimeout(ctx, 10*time.Second)
	defer clientCancel()

	// TODO Sanitize input: if it contains leading spaces, this crashes.
	c, err := rpc.DialContext(clientCtx, url)
	if err != nil {
		mod.LogErrorf("failed to start client: %v", err)
		return
	}
	defer c.Close()

	mod.Logf("connected")

	// Channel for events from the JSONRPC endpoint.
	subChan := make(chan map[string]string)
	// Channel for errors from the subscription goroutine.
	errChan := make(chan error, 1)

	go func() {
		if err := subscribeHeads(ctx, c, subChan); err != nil {
			errChan <- err
		}
	}()

	for {
		select {
		case ev := <-subChan:
			if err := sendEvent(mod.Name(), ev, eventChan); err != nil {
				mod.LogErrorf("failed sending event: %v", err)
				return
			}
		case err := <-errChan:
			mod.LogErrorf("subscription error: %v", err)
			return
		case <-mod.SigChan:
			return
		}
	}
}

func (mod *EthereumBee) ReloadOptions(options bees.BeeOptions) {
	mod.SetOptions(options)
}

func subscribeHeads(ctx context.Context, client *rpc.Client, ch chan map[string]string) error {
	sub, err := client.EthSubscribe(ctx, ch, "newHeads")
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-sub.Err():
		return err
	}
}

func sendEvent(bee string, ev map[string]string, eventChan chan bees.Event) error {
	num, err := hexutil.DecodeUint64(ev["number"])
	if err != nil {
		return err
	}

	tmp, err := hexutil.DecodeUint64(ev["timestamp"])
	if err != nil {
		return err
	}
	ts := int64(tmp)

	nonce, err := hexutil.DecodeUint64(ev["nonce"])
	if err != nil {
		return err
	}

	event := bees.Event{
		Bee:  bee,
		Name: "new_block",
		Options: []bees.Placeholder{
			{
				Name:  "number",
				Type:  "uint64",
				Value: num,
			},
			{
				Name:  "difficulty",
				Type:  "string",
				Value: ev["difficulty"],
			},
			{
				Name:  "miner",
				Type:  "string",
				Value: ev["miner"],
			},
			{
				Name:  "parentHash",
				Type:  "string",
				Value: ev["parentHash"],
			},
			{
				Name:  "timestamp",
				Type:  "string",
				Value: time.Unix(ts, 0),
			},
			{
				Name:  "nonce",
				Type:  "string",
				Value: nonce,
			},
		},
	}

	eventChan <- event

	return nil
}
