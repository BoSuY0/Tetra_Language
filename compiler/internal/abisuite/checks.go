package abisuite

import "fmt"

type Check struct {
	Name  string
	Error string
}

type Case struct {
	Name string
	Run  func() error
}

func RunChecks(cases []Case) []Check {
	out := make([]Check, 0, len(cases))
	for _, tc := range cases {
		check := Check{Name: tc.Name}
		if err := tc.Run(); err != nil {
			check.Error = err.Error()
		}
		out = append(out, check)
	}
	return out
}

func UnsupportedTargetError(target string) error {
	return fmt.Errorf("ABI suite for target %s is not implemented", target)
}
