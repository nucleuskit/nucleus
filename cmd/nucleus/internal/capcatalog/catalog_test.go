package capcatalog

import (
	"testing"

	"github.com/nucleuskit/contract/inspect"
)

func TestCatalogMatchesContractCapabilityModules(t *testing.T) {
	for _, spec := range All() {
		if module := inspect.CapabilityModule(spec.Name); module == "" {
			t.Fatalf("capability %q is missing a contract inspect module mapping", spec.Name)
		}
		if spec.DefaultProvider == "" {
			t.Fatalf("capability %q has empty default provider", spec.Name)
		}
		if !ProviderSupported(spec, spec.DefaultProvider) {
			t.Fatalf("capability %q default provider %q is not in provider list %#v", spec.Name, spec.DefaultProvider, spec.ProviderNames())
		}
	}
}

func TestPlanningNamesAreCatalogCapabilities(t *testing.T) {
	for _, name := range PlanningNames() {
		spec, ok := Lookup(name)
		if !ok {
			t.Fatalf("planning capability %q is not in catalog", name)
		}
		if !spec.Planning {
			t.Fatalf("planning capability %q has Planning=false", name)
		}
	}
}

func TestLookupNormalizesNameAndProvider(t *testing.T) {
	spec, ok := Lookup(" Redis ")
	if !ok {
		t.Fatal("Lookup Redis = false, want true")
	}
	if spec.Name != "redis" {
		t.Fatalf("spec.Name = %q, want redis", spec.Name)
	}
	if _, ok := spec.Provider(" GoRedis "); !ok {
		t.Fatalf("Provider GoRedis = false, want true")
	}
}
