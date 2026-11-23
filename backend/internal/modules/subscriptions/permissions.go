package subscriptions

import "mauzo/internal/core/authz"

// Subscription permissions (platform).
const (
	PermSubscriptionView   = "platform.subscription.view"
	PermSubscriptionManage = "platform.subscription.manage"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermSubscriptionView, Description: "View plans and subscriptions"},
		authz.Permission{Code: PermSubscriptionManage, Description: "Create plans and assign subscriptions"},
	)
}
