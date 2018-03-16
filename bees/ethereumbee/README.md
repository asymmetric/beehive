# Ethereum bee

Connects to an Ethereum JSONRPC endpoint, creating events when:

* a block is mined
* an (optional) address' balance changes

## configuration

* url: WebSocket URL of a JSONRPC endpoint.
* (optional) address: an Ethereum address

## events

**new_block**

* number: the number of the newly mined block.

**new_transaction**

* tx_id: the transaction id

## credits

logo: https://ethereum.org/assets
