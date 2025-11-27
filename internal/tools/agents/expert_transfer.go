package agents
import (
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"prometheus/pkg/errors"
)
// TransferToExpertTool creates a tool for explicit agent transfer to experts
func TransferToExpertTool() tool.Tool {
	t, _ := functiontool.New(
		functiontool.Config{
			Name:        "transfer_to_expert",
			Description: "Transfer control to a specialized expert agent when deeper analysis is needed in a specific domain (macro, onchain, derivatives, correlation)",
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			expertType, ok := args["expert"].(string)
			if !ok {
				return nil, errors.Wrap(errors.ErrInvalidInput, "expert type is required")
			}
			reason, ok := args["reason"].(string)
			if !ok {
				reason = "requested by agent"
			}
			// Validate expert type
			validExperts := map[string]string{
				"macro":       "MacroAnalyst",
				"onchain":     "OnChainAnalyst",
				"derivatives": "DerivativesAnalyst",
				"correlation": "CorrelationAnalyst",
			}
			targetAgent, valid := validExperts[expertType]
			if !valid {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "invalid expert type: %s. Valid types: macro, onchain, derivatives, correlation", expertType)
			}
			// Set transfer action via ADK EventActions
			if actions := ctx.Actions(); actions != nil {
				actions.TransferToAgent = targetAgent
			}
			return map[string]interface{}{
				"transferred": true,
				"target":      targetAgent,
				"reason":      reason,
				"message":     "Control transferred to " + targetAgent + ". They will provide specialized analysis and return control when complete.",
			}, nil
		},
	)
	return t
}


