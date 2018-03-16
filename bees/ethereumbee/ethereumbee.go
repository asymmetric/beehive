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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
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

	// TODO Sanitize input: if it contains leading spaces, this crashes.
	c, err := ethclient.Dial(url)
	if err != nil {
		mod.LogErrorf("failed to start client: %v", err)
		return
	}

	mod.Logf("connected")

	// Channel for events from the JSONRPC endpoint.
	subChan := make(chan *types.Header)
	// Channel for balance change events.
	blockNumChan := make(chan *big.Int)
	balChan := make(chan *big.Int)
	// Channel for errors from the subscription goroutine.
	errChan := make(chan error, 1)

	go func() {
		if err := subscribeHeads(ctx, c, subChan); err != nil {
			errChan <- err
		}
	}()

	if addr := mod.Options().Value("address"); addr != nil {
		addr := addr.(string)

		if !common.IsHexAddress(addr) {
			errChan <- err
			return
		}

		mod.Logf("Watching address %s\n", addr)

		go func() {
			if err := pollBalance(ctx, c, blockNumChan, balChan, addr); err != nil {
				errChan <- err
				return
			}
		}()
	}

	for {
		select {
		case h := <-subChan:
			if err := sendEvent(mod.Name(), h, eventChan); err != nil {
				mod.LogErrorf("failed sending event: %v", err)
				// TODO need to close the RPC connection!
				return
			}

			// TODO only do this if someone is consuming.
			blockNumChan <- h.Number
		case bal := <-balChan:
			mod.Logf("received balance: %s", bal.String())
			// TODO emit event
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

func subscribeHeads(ctx context.Context, client *ethclient.Client, ch chan *types.Header) error {
	sub, err := client.SubscribeNewHead(ctx, ch)
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

// pollBalance performs a `BalanceAt` call every time a message is receive on the `blockNum` channel.
// It then compares the current balance with the last one, and sends a message
// on the `balChan` channel if the new one is greater than the last.
func pollBalance(ctx context.Context, client *ethclient.Client, blockNum, balChan chan *big.Int, address string) error {
	addr := common.HexToAddress(address)
	lastNum := <-blockNum
	lastBal := big.NewInt(0)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case lastNum = <-blockNum:
			bal, err := client.BalanceAt(ctx, addr, lastNum)
			if err != nil {
				// Due to intermittent error responses from the endpoint, we simply log
				// and repeat the request in case of errors.
				fmt.Printf("error requesting balance: %v\n", err)
				continue
			}

			// If bal > lastBal.
			if bal.Cmp(lastBal) == 1 {
				lastBal = bal
				balChan <- bal
			}
		}
	}

	return nil
}

func sendEvent(bee string, h *types.Header, eventChan chan bees.Event) error {
	event := bees.Event{
		Bee:  bee,
		Name: "new_block",
		Options: []bees.Placeholder{
			{
				Name:  "number",
				Type:  "string",
				Value: h.Number.String(),
			},
			{
				Name:  "difficulty",
				Type:  "string",
				Value: h.Difficulty.String(),
			},
			{
				Name:  "miner",
				Type:  "string",
				Value: h.Coinbase.String(),
			},
			{
				Name:  "parentHash",
				Type:  "string",
				Value: h.ParentHash.String(),
			},
			{
				Name:  "timestamp",
				Type:  "string",
				Value: h.Time.String(),
			},
		},
	}

	eventChan <- event

	return nil
}
