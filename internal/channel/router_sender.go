package channel

import "context"

// RouterSender adapts channel.Router to the channel.Sender interface.
// While Router already satisfies Sender (same Send signature), this explicit
// adapter makes the intent clear and allows the two to diverge independently.
type RouterSender struct {
	router *Router
}

// NewRouterSender creates a new RouterSender wrapping the given Router.
func NewRouterSender(r *Router) *RouterSender {
	return &RouterSender{router: r}
}

// Send sends a message to a user identified by channel type + user ID.
func (rs *RouterSender) Send(ctx context.Context, channelType Type, userID string, text string) error {
	return rs.router.Send(ctx, channelType, userID, text)
}
