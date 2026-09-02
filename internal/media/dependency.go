package media

import (
	"context"
	"errors"
)

type Capability struct {
	Name         string
	Available    bool
	Incompatible bool
	Executable   string
	Version      string
	Operations   []Operation
}
type Platform struct{ GOOS, GOARCH string }
type DiagnosticState string

const (
	DiagnosticAvailable    DiagnosticState = "available"
	DiagnosticMissing      DiagnosticState = "missing"
	DiagnosticIncompatible DiagnosticState = "incompatible"
	DiagnosticUnsupported  DiagnosticState = "unsupported"
)

type InstallRecipe struct {
	Dependency, GOOS, GOARCH string
	Command                  []string
	Instructions             string
}
type Diagnostic struct {
	Dependency     string
	State          DiagnosticState
	InstallCommand []string
	Instructions   string
}
type CapabilityProbe interface {
	Probe(context.Context, string) (Capability, error)
}

func Discover(ctx context.Context, dependency string, probe CapabilityProbe) (Capability, error) {
	return probe.Probe(ctx, dependency)
}
func Diagnose(capability Capability, platform Platform, recipes []InstallRecipe) Diagnostic {
	diagnostic := Diagnostic{Dependency: capability.Name}
	if capability.Available {
		if capability.Incompatible {
			diagnostic.State = DiagnosticIncompatible
		} else {
			diagnostic.State = DiagnosticAvailable
		}
		return diagnostic
	}
	for _, recipe := range recipes {
		if recipe.Dependency == capability.Name && recipe.GOOS == platform.GOOS && recipe.GOARCH == platform.GOARCH {
			diagnostic.State, diagnostic.InstallCommand, diagnostic.Instructions = DiagnosticMissing, cloneStrings(recipe.Command), recipe.Instructions
			return diagnostic
		}
	}
	diagnostic.State = DiagnosticUnsupported
	return diagnostic
}

type SeparateConsent struct{ ModelDownload, Credentials, LicenseAcceptance, RemoteTransfer bool }
type RemediationRequest struct {
	Dependency             string
	Platform               Platform
	Interactive, Confirmed bool
	Consent                SeparateConsent
}
type Installer interface {
	Install(context.Context, []string) error
}
type Discoverer interface {
	Discover(context.Context, string) (Capability, error)
}
type RemediationResult struct {
	Command       []string
	Before, After Capability
}

var ErrConfirmationRequired = errors.New("explicit installation confirmation required")
var ErrNoKnownRecipe = errors.New("no known installation recipe")

func Remediate(ctx context.Context, request RemediationRequest, recipes []InstallRecipe, installer Installer, discoverer Discoverer) (RemediationResult, error) {
	before, err := discoverer.Discover(ctx, request.Dependency)
	if err != nil {
		return RemediationResult{}, err
	}
	result := RemediationResult{Before: before}
	var recipe *InstallRecipe
	for i := range recipes {
		if recipes[i].Dependency == request.Dependency && recipes[i].GOOS == request.Platform.GOOS && recipes[i].GOARCH == request.Platform.GOARCH {
			recipe = &recipes[i]
			break
		}
	}
	if recipe == nil {
		return result, ErrNoKnownRecipe
	}
	result.Command = cloneStrings(recipe.Command)
	if !request.Interactive || !request.Confirmed {
		return result, ErrConfirmationRequired
	}
	if err := installer.Install(ctx, cloneStrings(recipe.Command)); err != nil {
		return result, err
	}
	after, err := discoverer.Discover(ctx, request.Dependency)
	result.After = after
	return result, err
}
func cloneStrings(values []string) []string { return append([]string(nil), values...) }
