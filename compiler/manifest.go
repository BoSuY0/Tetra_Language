package compiler

import (
	"fmt"

	"tetra_language/compiler/internal/formats"
	"tetra_language/compiler/internal/semantics"
	ctarget "tetra_language/compiler/target"
)

type Manifest struct {
	CompilerVersion string            `json:"compiler_version"`
	Formats         []FormatManifest  `json:"formats"`
	Targets         []TargetManifest  `json:"targets"`
	Builtins        []BuiltinManifest `json:"builtins"`
	RuntimeABI      RuntimeManifest   `json:"runtime_abi"`
}

type FormatManifest = formats.Info

type TargetManifest struct {
	Triple         string `json:"triple"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	ABI            string `json:"abi"`
	Format         string `json:"format"`
	ExeExt         string `json:"exe_ext"`
	CollectImports bool   `json:"collect_imports"`
}

type BuiltinManifest struct {
	Name          string   `json:"name"`
	Aliases       []string `json:"aliases,omitempty"`
	ParamTypes    []string `json:"param_types,omitempty"`
	ReturnType    string   `json:"return_type"`
	Effects       []string `json:"effects,omitempty"`
	UnsafePolicy  string   `json:"unsafe_policy"`
	UnsafeDetails string   `json:"unsafe_details,omitempty"`
}

type RuntimeManifest struct {
	ReservedPrefix           string   `json:"reserved_prefix"`
	ActorsSupportedTargets   []string `json:"actors_supported_targets"`
	ActorsRequiredSymbols    []string `json:"actors_required_symbols"`
	TimeRequiredSymbols      []string `json:"time_required_symbols,omitempty"`
	ActorsProgramGlueSymbols []string `json:"actors_program_glue_symbols"`
}

func GetManifest() (Manifest, error) {
	builtins, err := semantics.DescribeBuiltins()
	if err != nil {
		return Manifest{}, err
	}
	builtinOut := make([]BuiltinManifest, 0, len(builtins))
	for _, b := range builtins {
		builtinOut = append(builtinOut, BuiltinManifest{
			Name:          b.Name,
			Aliases:       append([]string(nil), b.Aliases...),
			ParamTypes:    append([]string(nil), b.ParamTypes...),
			ReturnType:    b.ReturnType,
			Effects:       append([]string(nil), b.Effects...),
			UnsafePolicy:  b.UnsafePolicy,
			UnsafeDetails: b.UnsafeDetails,
		})
	}

	targets := ctarget.All()
	targetOut := make([]TargetManifest, 0, len(targets))
	for _, t := range targets {
		targetOut = append(targetOut, TargetManifest{
			Triple:         t.Triple,
			OS:             fmt.Sprint(t.OS),
			Arch:           fmt.Sprint(t.Arch),
			ABI:            fmt.Sprint(t.ABI),
			Format:         fmt.Sprint(t.Format),
			ExeExt:         t.ExeExt,
			CollectImports: t.CollectImports,
		})
	}

	return Manifest{
		CompilerVersion: Version(),
		Formats:         formats.All(),
		Targets:         targetOut,
		Builtins:        builtinOut,
		RuntimeABI: RuntimeManifest{
			ReservedPrefix:         "__tetra_",
			ActorsSupportedTargets: []string{"linux-x64", "macos-x64", "windows-x64"},
			ActorsRequiredSymbols:  requiredActorRuntimeSymbols(),
			TimeRequiredSymbols:    requiredTimeRuntimeSymbols(),
			ActorsProgramGlueSymbols: []string{
				"__tetra_actor_dispatch",
				"__tetra_actor_main_entry_id",
			},
		},
	}, nil
}
