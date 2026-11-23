package dashboard

import (
	"sort"

	"mauzo/internal/core/identity"
	"mauzo/internal/core/modules"
)

// Widgets returns the dashboard widgets the actor may see: widgets of enabled
// modules whose permission the actor holds, in layout order. Modules declare
// their widgets in the module catalogue; the web app maps each key to a
// component, so a module "plugs in" by declaring a widget and shipping its
// component and data endpoint.
func Widgets(a identity.Actor) []modules.Widget {
	out := []modules.Widget{}
	for _, m := range modules.All() {
		if !a.HasModule(m.Code) {
			continue
		}
		for _, w := range m.Widgets {
			if w.Permission == "" || a.Can(w.Permission) {
				out = append(out, w)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out
}
