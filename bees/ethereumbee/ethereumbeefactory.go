package ethereumbee

import "github.com/muesli/beehive/bees"

type EthereumBeeFactory struct {
	bees.BeeFactory
}

func (factory *EthereumBeeFactory) New(name, description string, options bees.BeeOptions) bees.BeeInterface {
	bee := EthereumBee{
		Bee: bees.NewBee(name, factory.ID(), description, options),
	}

	bee.ReloadOptions(options)

	return &bee
}

func (factory *EthereumBeeFactory) ID() string {
	return "ethereumbee"
}

// Name returns the name of this Bee.
func (factory *EthereumBeeFactory) Name() string {
	return "Ethereum JSONRPC"
}

// Description returns the desciption of this Bee.
func (factory *EthereumBeeFactory) Description() string {
	return "Reacts to events on the Ethereum blockchain"
}

// Image returns the filename of an image for this Bee.
func (factory *EthereumBeeFactory) Image() string {
	return factory.ID() + ".png"
}

// LogoColor returns ther preferred logo background color (used by the admin interface).
func (factory *EthereumBeeFactory) LogoColor() string {
	return "#6098d0"
}

// Options returns the options available to configure this Bee.
func (factory *EthereumBeeFactory) Options() []bees.BeeOptionDescriptor {
	return []bees.BeeOptionDescriptor{
		{
			Name:        "url",
			Description: "The JSONRPC WebSocket URL",
			Type:        "string",
			Mandatory:   true,
		},
		{
			Name:        "address",
			Description: "The address to watch",
			Type:        "string",
			Mandatory:   false,
		},
	}
}

// Events describes the available events provided by this Bee.
func (factory *EthereumBeeFactory) Events() []bees.EventDescriptor {
	return []bees.EventDescriptor{
		{
			Namespace:   factory.Name(),
			Name:        "new_block",
			Description: "A new block was mined",
			Options: []bees.PlaceholderDescriptor{
				{
					Name:        "number",
					Description: "The block number",
					Type:        "string",
				},
				{
					Name:        "difficulty",
					Description: "The block difficulty",
					Type:        "string",
				},
				{
					Name:        "miner",
					Description: "The address of the miner of the block",
					Type:        "string",
				},
				{
					Name:        "parentHash",
					Description: "The block's parent hash",
					Type:        "string",
				},
				{
					Name:        "timestamp",
					Description: "The block timestamp",
					Type:        "string",
				},
				{
					Name:        "nonce",
					Description: "The block nonce",
					Type:        "string",
				},
			},
		},
		{
			Namespace:   factory.Name(),
			Name:        "new_transaction",
			Description: "A transaction involving the address was mined",
			Options: []bees.PlaceholderDescriptor{
				{
					Name:        "txid",
					Description: "The transaction ID",
					Type:        "string",
				},
			},
		},
	}
}

func init() {
	f := EthereumBeeFactory{}
	bees.RegisterFactory(&f)
}
