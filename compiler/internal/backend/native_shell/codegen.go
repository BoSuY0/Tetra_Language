package native_shell

import (
	"strings"

	"tetra_language/compiler/internal/lower"
)

func Render(bundle *lower.UILoweredBundle) []byte {
	if bundle == nil {
		return []byte("Tetra Native UI Shell\n(no UI metadata)\n")
	}
	lines := []string{
		"Tetra Native UI Shell",
		"schema: " + bundle.Schema,
	}
	for _, view := range bundle.Views {
		lines = append(lines, "")
		lines = append(lines, "view "+view.Name+" (state: "+view.StateType+")")
		for _, binding := range view.Bindings {
			lines = append(lines, "  bind "+binding.Name+": "+binding.Type+" <- "+binding.Source)
		}
		for _, event := range view.Events {
			lines = append(lines, "  event "+event.Name+" -> "+event.Command)
		}
		for _, cmd := range view.Commands {
			lines = append(lines, "  command "+cmd.Name+" ("+itoa(cmd.StatementCount)+" stmt)")
		}
		for _, style := range view.Styles {
			lines = append(lines, "  style "+style.Name+": "+style.Type+" = "+style.Value)
		}
		for _, entry := range view.Accessibility {
			lines = append(lines, "  accessibility "+entry.Name+": "+entry.Type+" = "+entry.Value)
		}
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
