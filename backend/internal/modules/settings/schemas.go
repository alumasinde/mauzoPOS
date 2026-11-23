package settings

import coresettings "mauzo/internal/core/settings"

// Registered settings schemas. Every key stored in tenant_settings must be
// listed here — unknown keys are rejected. Modules that add settings keys
// register their own schemas in their package init.
func init() {
	coresettings.RegisterSchema("business.currency", map[string]any{
		"type": "string", "minLength": 3, "maxLength": 3})
	coresettings.RegisterSchema("tax.prices_include_vat", map[string]any{
		"type": "boolean"})
	coresettings.RegisterSchema("tax.default_rate", map[string]any{
		"type": "number", "enum": []any{0, 16}})
	coresettings.RegisterSchema("payments.methods", map[string]any{
		"type": "array",
		"items": map[string]any{"type": "string",
			"enum": []any{"cash", "card", "mpesa", "bank", "credit"}}})
	coresettings.RegisterSchema("receipt.footer", map[string]any{
		"type": "string", "maxLength": 500})
	coresettings.RegisterSchema("receipt.paper_width", map[string]any{
		"type": "integer", "enum": []any{58, 80}})
	coresettings.RegisterSchema("receipt.mpesa_till", map[string]any{
		"type": "string", "maxLength": 20})
	coresettings.RegisterSchema("receipt.mpesa_paybill", map[string]any{
		"type": "string", "maxLength": 20})
	coresettings.RegisterSchema("receipt.mpesa_account", map[string]any{
		"type": "string", "maxLength": 40})
	coresettings.RegisterSchema("receipt.auto_print", map[string]any{
		"type": "boolean"})
	coresettings.RegisterSchema("receipt.copies", map[string]any{
		"type": "integer", "enum": []any{1, 2, 3}})
	coresettings.RegisterSchema("receipt.open_drawer", map[string]any{
		"type": "boolean"})
	coresettings.RegisterSchema("profile.receipt_template", map[string]any{
		"type": "string", "maxLength": 20000})
	coresettings.RegisterSchema("profile.pos_layout", map[string]any{
		"type": "object"})
	coresettings.RegisterSchema("catalog.default_categories", map[string]any{
		"type": "array", "items": map[string]any{"type": "string", "maxLength": 60}})
	coresettings.RegisterSchema("pos.scale_barcode_prefix", map[string]any{
		"type": "array", "items": map[string]any{"type": "string", "minLength": 2, "maxLength": 3}})
}
