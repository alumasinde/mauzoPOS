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
// shown when the module is enabled and the user holds Permission.
type NavItem struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Path       string `json:"path"`
	Icon       string `json:"icon"`
	Permission string `json:"-"`
	Order      int    `json:"-"`
}

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
	// Nav entries (only for screens that exist).
	Nav []NavItem
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

// validateGraph panics on unknown dependencies or cycles (boot-time check).
func validateGraph() {
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
