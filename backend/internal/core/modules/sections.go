package modules

// NavSection is a heading in the back-office sidebar. Sections are listed in
// display order; a section with no visible items is not shown.
type NavSection struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Sidebar section keys.
const (
	SectionTop        = "" // above every heading (Dashboard)
	SectionSales      = "sales"
	SectionInventory  = "inventory"
	SectionPricing    = "pricing"
	SectionPurchasing = "purchasing"
	SectionPayments   = "payments"
	SectionReports    = "reports"
	SectionTeam       = "team"
	SectionConfig     = "configuration"
)

var sections = []NavSection{
	{Key: SectionTop},
	{Key: SectionSales, Label: "Sales"},
	{Key: SectionInventory, Label: "Inventory"},
	{Key: SectionPricing, Label: "Pricing"},
	{Key: SectionPurchasing, Label: "Purchasing"},
	{Key: SectionPayments, Label: "Payments"},
	{Key: SectionReports, Label: "Reports"},
	{Key: SectionTeam, Label: "Team & access"},
	{Key: SectionConfig, Label: "Configuration"},
}

// Sections returns the sidebar sections in display order.
func Sections() []NavSection { return append([]NavSection(nil), sections...) }

// SectionByKey returns a section and its display position.
func SectionByKey(key string) (int, bool) {
	for i, s := range sections {
		if s.Key == key {
			return i, true
		}
	}
	return 0, false
}
