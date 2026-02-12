package events

import (
	"context"
	"fmt"
)

type Dispatcher struct {
	GitHub *GitHubDispatcher
}

func NewDispatcher(github *GitHubDispatcher) *Dispatcher {
	return &Dispatcher{
		GitHub: github,
	}
}

func (d *Dispatcher) Dispatch(
	ctx context.Context,
	event *RuntimeEvent,
) error {

	if event == nil {
		return fmt.Errorf("runtime event is nil")
	}

	// switch event.Source {
	// case SourceGitHub:
	// 	return d.GitHub.Handle(ctx, event)

	// default:
	// 	return fmt.Errorf("unsupported event source %q", event.Source)
	// }
	return nil
}
