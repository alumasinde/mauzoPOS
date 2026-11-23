// Package dashboard serves the back-office dashboard layout: the widgets the
// current user may see, contributed by enabled modules (see Widgets).
package dashboard

import "mauzo/internal/core/authz"

// PermView allows viewing the business dashboard.
const PermView = "dashboard.view"

func init() {
	authz.Define(authz.Permission{Code: PermView, Module: "dashboard", Description: "View the business dashboard"})
}
