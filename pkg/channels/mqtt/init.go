package mqtt

import (
	"github.com/oryna-ai/picoclaw/pkg/bus"
	"github.com/oryna-ai/picoclaw/pkg/channels"
	"github.com/oryna-ai/picoclaw/pkg/config"
)

func init() {
	channels.RegisterSafeFactory(
		config.ChannelMQTT,
		func(bc *config.Channel, cfg *config.MQTTSettings, b *bus.MessageBus) (channels.Channel, error) {
			return NewMQTTChannel(bc, cfg, b)
		},
	)
}
