package agent

// HookMeta contains correlation fields shared by agent hook requests and
// runtime events emitted from turn processing.
type HookMeta struct {
	AgentID      string
	TurnID       string
	StateID      string
	ParentTurnID string
	SessionKey   string
	Iteration    int
	TracePath    string
	Source       string
	Channel      string
	ChatID       string
	turnContext  *TurnContext
}
