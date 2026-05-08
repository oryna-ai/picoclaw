// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
)

// outboundHookAdapter adapts HookManager to channels.OutboundHook.
type outboundHookAdapter struct {
	hm *HookManager
}

// NewOutboundHookAdapter creates a channels.OutboundHook from a HookManager.
// This allows the agent's hook system to intercept outbound messages.
func NewOutboundHookAdapter(hm *HookManager) channels.OutboundHook {
	if hm == nil {
		return nil
	}
	return &outboundHookAdapter{hm: hm}
}

func (a *outboundHookAdapter) buildMeta(name string, msg *bus.OutboundMessage) HookMeta {
	return HookMeta{
		AgentID:    msg.AgentID,
		TurnID:     msg.TurnID,
		StateID:    msg.StateID,
		SessionKey: msg.SessionKey,
		Channel:    name,
		ChatID:     msg.ChatID,
		turnContext: &TurnContext{
			Inbound: &msg.Context,
		},
	}
}

// BeforeOutbound is a pass-through — MessageInterceptor only has AfterOutbound.
func (a *outboundHookAdapter) BeforeOutbound(
	_ context.Context,
	_ string,
	msg *bus.OutboundMessage,
) (*bus.OutboundMessage, bool, error) {
	return msg, false, nil
}

func (a *outboundHookAdapter) AfterOutbound(
	ctx context.Context,
	name string,
	msg *bus.OutboundMessage,
	msgIDs []string,
	sendErr error,
) {
	if a == nil || a.hm == nil || msg == nil {
		return
	}

	errStr := ""
	if sendErr != nil {
		errStr = sendErr.Error()
	}

	resp := &OutboundHookResponse{
		Meta:       a.buildMeta(name, msg),
		Context:    &TurnContext{Inbound: &msg.Context},
		Message:    *msg,
		MessageIDs: msgIDs,
		Error:      errStr,
	}

	a.hm.AfterOutbound(ctx, resp)
}
