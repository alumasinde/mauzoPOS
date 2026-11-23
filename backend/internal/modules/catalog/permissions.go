package catalog

import "mauzo/internal/core/authz"

// Catalog permissions.
const (
	PermProductView   = "catalog.product.view"
	PermProductCreate = "catalog.product.create"
	PermProductUpdate = "catalog.product.update"
	PermProductDelete = "catalog.product.delete"
	PermProductImport = "catalog.product.import"
	PermProductExport = "catalog.product.export"
	PermTaxonomy      = "catalog.category.manage"
)

func init() {
	authz.Define(
		authz.Permission{Code: PermProductView, Module: "catalog", Description: "View products"},
		authz.Permission{Code: PermProductCreate, Module: "catalog", Description: "Create products"},
		authz.Permission{Code: PermProductUpdate, Module: "catalog", Description: "Edit products"},
		authz.Permission{Code: PermProductDelete, Module: "catalog", Description: "Archive products"},
		authz.Permission{Code: PermProductImport, Module: "catalog", Description: "Bulk-import products"},
		authz.Permission{Code: PermProductExport, Module: "catalog", Description: "Export the catalog"},
		authz.Permission{Code: PermTaxonomy, Module: "catalog", Description: "Manage categories, brands and units"},
	)
}
