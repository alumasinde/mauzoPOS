package subscriptions

import "mauzo/internal/core/authz"

// Plan & subscription permissions.
const (
	PermSubscriptionView   = "platform.subscription.view"
	PermSubscriptionManage = "platform.subscription.manage"
)

func init() {
	authz.Register(PermSubscriptionView, PermSubscriptionManage)
}
