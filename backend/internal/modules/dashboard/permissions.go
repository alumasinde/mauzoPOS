// Package dashboard will serve the configurable back-office dashboard
// widgets. For now it owns the dashboard permission.
package dashboard

import "mauzo/internal/core/authz"

// PermView allows viewing the business dashboard.
const PermView = "dashboard.view"

func init() {
	authz.Define(authz.Permission{Code: PermView, Module: "dashboard", Description: "View the business dashboard"})
}
