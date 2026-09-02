package media

import (
	"context"
	"reflect"
	"testing"
)

func TestDoctorDiagnosticStatesAndKnownRecipes(t *testing.T) {
	recipes := []InstallRecipe{{Dependency: "parakeet", GOOS: "darwin", GOARCH: "arm64", Command: []string{"brew", "install", "parakeet"}}}
	tests := []struct {
		name        string
		cap         Capability
		platform    Platform
		want        DiagnosticState
		wantCommand []string
	}{
		{"available", Capability{Name: "parakeet", Available: true}, Platform{"darwin", "arm64"}, DiagnosticAvailable, nil},
		{"known recipe", Capability{Name: "parakeet"}, Platform{"darwin", "arm64"}, DiagnosticMissing, []string{"brew", "install", "parakeet"}},
		{"unsupported", Capability{Name: "parakeet"}, Platform{"plan9", "mips"}, DiagnosticUnsupported, nil},
		{"incompatible", Capability{Name: "parakeet", Available: true, Incompatible: true}, Platform{"darwin", "arm64"}, DiagnosticIncompatible, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Diagnose(tt.cap, tt.platform, recipes)
			if got.State != tt.want || !reflect.DeepEqual(got.InstallCommand, tt.wantCommand) {
				t.Fatalf("diagnostic mismatch: %#v", got)
			}
		})
	}
}

func TestDiscoverReportsIdentityVersionAndOperations(t *testing.T) {
	probe := &fakeProbe{capability: Capability{Name: "parakeet", Available: true, Executable: "/opt/parakeet", Version: "0.1", Operations: []Operation{OperationTranscribe}}}
	got, err := Discover(context.Background(), "parakeet", probe)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, probe.capability) || probe.calls != 1 {
		t.Fatalf("discovery mismatch: %#v calls=%d", got, probe.calls)
	}
}

func TestRemediationRequiresExactCommandAndConfirmation(t *testing.T) {
	recipe := InstallRecipe{Dependency: "parakeet", GOOS: "darwin", GOARCH: "arm64", Command: []string{"brew", "install", "parakeet"}}
	installer := &fakeInstaller{}
	discoverer := &sequenceDiscoverer{results: []Capability{{Name: "parakeet"}, {Name: "parakeet"}, {Name: "parakeet", Available: true}}}

	result, err := Remediate(context.Background(), RemediationRequest{Dependency: "parakeet", Platform: Platform{"darwin", "arm64"}, Interactive: false}, []InstallRecipe{recipe}, installer, discoverer)
	if err != ErrConfirmationRequired {
		t.Fatalf("expected confirmation refusal, got %v", err)
	}
	if installer.calls != 0 || !reflect.DeepEqual(result.Command, recipe.Command) {
		t.Fatalf("must display exact command without install: %#v calls=%d", result, installer.calls)
	}

	result, err = Remediate(context.Background(), RemediationRequest{Dependency: "parakeet", Platform: Platform{"darwin", "arm64"}, Interactive: true, Confirmed: true}, []InstallRecipe{recipe}, installer, discoverer)
	if err != nil {
		t.Fatal(err)
	}
	if installer.calls != 1 || discoverer.calls != 3 || !result.After.Available {
		t.Fatalf("install/recheck mismatch: %#v installs=%d discovers=%d", result, installer.calls, discoverer.calls)
	}
}

func TestPackageConsentDoesNotGrantSeparateConsents(t *testing.T) {
	request := RemediationRequest{Dependency: "parakeet", Interactive: true, Confirmed: true}
	if request.Consent.ModelDownload || request.Consent.Credentials || request.Consent.LicenseAcceptance || request.Consent.RemoteTransfer {
		t.Fatalf("package confirmation leaked into separate consent: %#v", request.Consent)
	}
}

type fakeProbe struct {
	capability Capability
	calls      int
}

func (f *fakeProbe) Probe(context.Context, string) (Capability, error) {
	f.calls++
	return f.capability, nil
}

type fakeInstaller struct{ calls int }

func (f *fakeInstaller) Install(context.Context, []string) error { f.calls++; return nil }

type sequenceDiscoverer struct {
	results []Capability
	calls   int
}

func (f *sequenceDiscoverer) Discover(context.Context, string) (Capability, error) {
	r := f.results[f.calls]
	f.calls++
	return r, nil
}
