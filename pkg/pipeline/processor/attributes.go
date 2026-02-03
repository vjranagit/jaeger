package processor

import (
	"context"

	"github.com/vjranagit/jaeger-toolkit/pkg/model"
)

// AttributesProcessor modifies span attributes (tags)
type AttributesProcessor struct {
	name    string
	actions []AttributeAction
}

// AttributeAction represents a single attribute modification
type AttributeAction struct {
	Key    string
	Value  string
	Action ActionType
}

// ActionType specifies how to modify an attribute
type ActionType string

const (
	// Insert adds attribute if it doesn't exist
	Insert ActionType = "insert"
	// Update modifies attribute if it exists
	Update ActionType = "update"
	// Upsert adds or modifies attribute
	Upsert ActionType = "upsert"
	// Delete removes attribute
	Delete ActionType = "delete"
)

// AttributesConfig configures the attributes processor
type AttributesConfig struct {
	Actions []AttributeAction
}

// NewAttributesProcessor creates a new attributes processor
func NewAttributesProcessor(name string, config AttributesConfig) *AttributesProcessor {
	return &AttributesProcessor{
		name:    name,
		actions: config.Actions,
	}
}

// Process applies attribute modifications to spans
func (p *AttributesProcessor) Process(ctx context.Context, in <-chan *model.Span) <-chan *model.Span {
	out := make(chan *model.Span)

	go func() {
		defer close(out)

		for {
			select {
			case span, ok := <-in:
				if !ok {
					return
				}

				// Apply all actions to this span
				for _, action := range p.actions {
					p.applyAction(span, action)
				}

				select {
				case out <- span:
				case <-ctx.Done():
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

// applyAction applies a single action to a span
func (p *AttributesProcessor) applyAction(span *model.Span, action AttributeAction) {
	// Find existing tag
	existingIdx := -1
	for i, tag := range span.Tags {
		if tag.Key == action.Key {
			existingIdx = i
			break
		}
	}

	switch action.Action {
	case Insert:
		if existingIdx == -1 {
			span.Tags = append(span.Tags, model.KeyValue{
				Key:   action.Key,
				VType: model.StringType,
				VStr:  action.Value,
			})
		}

	case Update:
		if existingIdx >= 0 {
			span.Tags[existingIdx] = model.KeyValue{
				Key:   action.Key,
				VType: model.StringType,
				VStr:  action.Value,
			}
		}

	case Upsert:
		if existingIdx >= 0 {
			span.Tags[existingIdx] = model.KeyValue{
				Key:   action.Key,
				VType: model.StringType,
				VStr:  action.Value,
			}
		} else {
			span.Tags = append(span.Tags, model.KeyValue{
				Key:   action.Key,
				VType: model.StringType,
				VStr:  action.Value,
			})
		}

	case Delete:
		if existingIdx >= 0 {
			span.Tags = append(span.Tags[:existingIdx], span.Tags[existingIdx+1:]...)
		}
	}
}

// Name returns the processor name
func (p *AttributesProcessor) Name() string {
	return p.name
}
