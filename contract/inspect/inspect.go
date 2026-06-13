package inspect

import (
	"path/filepath"

	"github.com/nucleuskit/contract/errors"
	"github.com/nucleuskit/contract/manifest"
	"github.com/nucleuskit/contract/openapi"
	"github.com/nucleuskit/contract/proto"
)

// Describe loads contract and manifest metadata for a service directory.
func Describe(dir string) (Description, error) {
	m, err := manifest.Load(dir)
	if err != nil {
		return Description{}, err
	}
	endpoints, err := openapi.LoadEndpoints(dir)
	if err != nil {
		return Description{}, err
	}
	grpcServices, err := proto.LoadServices(dir)
	if err != nil {
		return Description{}, err
	}
	errorCodes, err := errors.Load(dir)
	if err != nil {
		return Description{}, err
	}
	imports, err := ImportGraph(dir)
	if err != nil {
		return Description{}, err
	}

	return Description{
		SchemaVersion:      descriptionSchemaVersion,
		Service:            m.Service,
		Capabilities:       m.Capabilities,
		Endpoints:          endpoints,
		GRPCServices:       grpcServices,
		ErrorCodes:         errorCodes,
		Dependencies:       m.Dependencies,
		Modules:            readModules(filepath.Join(dir, goModFileName)),
		ConfigKeys:         collectConfigKeys(dir),
		Policy:             defaultPolicy(),
		EditSurfaces:       editSurfaces(m),
		GeneratedFreshness: freshness(dir, m),
		CapabilityGraph:    capabilityGraph(m, imports),
		Verification:       defaultVerification(),
	}, nil
}

// defaultPolicy 默认策略
func defaultPolicy() map[string]any {
	return map[string]any{defaultPolicyOutboundKey: map[string]any{}}
}

// defaultVerification 默认验证
func defaultVerification() Verification {
	return Verification{
		Commands: []string{
			commandValidate,
			commandLint,
			commandGoTest,
		},
	}
}
