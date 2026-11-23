// Package modules is the module registry: the code-defined description of
// every feature module (dependencies, config schema, navigation, business-type
// recommendations). The platform-wide on/off switch lives in the modules
// table and a tenant's enablement in tenant_modules — both owned by
// internal/modules/registry. Feature code never checks business types; it
// asks whether a module is enabled.
package modules

import (
	"fmt"
	"slices"
)

// Group buckets modules in the UI.
type Group string

// Module groups.
const (
	GroupCore       Group = "core"
	GroupOperations Group = "operations"
	GroupFinance    Group = "finance"
	GroupIndustry   Group = "industry"
	GroupGrowth     Group = "growth"
)

// NavItem is one back-office sidebar entry contributed by a module. It is
// shown when the module is enabled and the user holds Permission. Entries may
// point at screens the frontend has not shipped yet: the web app shows those
// as "coming soon" until the page exists, so the sidebar keeps its final shape
// as modules are built.
type NavItem struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Path       string `json:"path"` // may carry a query, e.g. /inventory?tab=movements
	Icon       string `json:"icon"`
	Section    string `json:"section"` // a NavSection key; "" = top of the sidebar
	Permission string `json:"-"`
	Order      int    `json:"-"` // position within the section
}

// Widget is one dashboard card contributed by a module. Like NavItem it is
// shown when the module is enabled and the user holds Permission; the web app
// maps Key to a component and renders a placeholder until that exists.
type Widget struct {
	Key        string     `json:"key"`
	Title      string     `json:"title"`
	Size       WidgetSize `json:"size"`
	Permission string     `json:"-"`
	Order      int        `json:"order"`
}

// WidgetSize controls where and how wide a widget is laid out.
type WidgetSize string

// Widget sizes.
const (
	WidgetKPI    WidgetSize = "kpi"    // headline figure in the top row
	WidgetWide   WidgetSize = "wide"   // two thirds of a panel row
	WidgetNarrow WidgetSize = "narrow" // one third of a panel row
)

// Module describes one enableable feature area.
type Module struct {
	Code        string
	Name        string
	Description string
	Group       Group
	// Core modules are always enabled for every tenant and cannot be switched
	// off by tenants, plans or the platform.
	Core bool
	// Depends lists module codes that must be enabled first.
	Depends []string
	// RecommendedFor lists business type codes this module suits. It is a hint
	// for onboarding/UI only — any tenant may enable any module.
	RecommendedFor []string
	// ConfigSchema validates tenant_modules.config (core/settings subset).
	ConfigSchema map[string]any
	// Nav entries, grouped into sidebar sections.
	Nav []NavItem
	// Widgets this module contributes to the dashboard.
	Widgets []Widget
}

var (
	byCode = map[string]Module{}
	order  []string
)

func register(mods ...Module) {
	for _, m := range mods {
		if _, dup := byCode[m.Code]; dup {
			panic(fmt.Sprintf("modules: duplicate module %q", m.Code))
		}
		byCode[m.Code] = m
		order = append(order, m.Code)
	}
}

// validateGraph panics on unknown dependencies or cycles, and on nav or widget
// entries that are malformed or duplicated (boot-time check).
func validateGraph() {
	navKeys, widgetKeys := map[string]bool{}, map[string]bool{}
	for _, code := range order {
		m := byCode[code]
		for _, n := range m.Nav {
			if _, ok := SectionByKey(n.Section); !ok {
				panic(fmt.Sprintf("modules: nav %q has unknown section %q", n.Key, n.Section))
			}
			if navKeys[n.Key] {
				panic(fmt.Sprintf("modules: duplicate nav key %q", n.Key))
			}
			navKeys[n.Key] = true
		}
		for _, w := range m.Widgets {
			switch w.Size {
			case WidgetKPI, WidgetWide, WidgetNarrow:
			default:
				panic(fmt.Sprintf("modules: widget %q has unknown size %q", w.Key, w.Size))
			}
			if widgetKeys[w.Key] {
				panic(fmt.Sprintf("modules: duplicate widget key %q", w.Key))
			}
			widgetKeys[w.Key] = true
		}
	}
	for _, code := range order {
		for _, d := range byCode[code].Depends {
			if _, ok := byCode[d]; !ok {
				panic(fmt.Sprintf("modules: %q depends on unknown %q", code, d))
			}
		}
	}
	state := map[string]int{} // 0 new, 1 visiting, 2 done
	var visit func(string, []string)
	visit = func(c string, path []string) {
		switch state[c] {
		case 1:
			panic(fmt.Sprintf("modules: dependency cycle %v", append(path, c)))
		case 2:
			return
		}
		state[c] = 1
		for _, d := range byCode[c].Depends {
			visit(d, append(path, c))
		}
		state[c] = 2
	}
	for _, c := range order {
		visit(c, nil)
	}
}

// Get returns a module by code.
func Get(code string) (Module, bool) {
	m, ok := byCode[code]
	return m, ok
}

// Known reports whether code is a registered module.
func Known(code string) bool {
	_, ok := byCode[code]
	return ok
}

// All returns every module in catalogue order.
func All() []Module {
	out := make([]Module, len(order))
	for i, c := range order {
		out[i] = byCode[c]
	}
	return out
}

// Codes returns every module code in catalogue order.
func Codes() []string { return slices.Clone(order) }

// CoreCodes returns the always-on modules.
func CoreCodes() []string {
	var out []string
	for _, c := range order {
		if byCode[c].Core {
			out = append(out, c)
		}
	}
	return out
}
