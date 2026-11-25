package main

import (
	"context"
	"errors"
	"mosaune/extensionkit/packageformat"
)

// Implemented only by trusted host adapters. Parsing must verify signature,
// adapter identity, schema and core compatibility before returning a package.
// Mutation methods must recheck expected checksums atomically; previews grant
// no rights. Shared catalogue data belongs to the host, not the package.
type metadataPackageProvider interface {
	parseProviderPackage([]byte) (packageformat.Package, error)
	packageSigner() string
	packageState(context.Context) (providerPackageState, error)
	installedPackage(context.Context) (packageformat.Package, error)
	installPackage(context.Context, packageformat.Package, string, bool) error
	rollbackPackage(context.Context, string, string) error
	removePackage(context.Context, string) error
}

var errProviderRollbackCorrupt = errors.New("Version précédente altérée.")

type providerPackageState struct {
	Registration        *providerRegistration  `json:"registration,omitempty"`
	RegistrationWarning string                 `json:"registrationWarning,omitempty"`
	Package             *packageformat.Package `json:"package"`
	PreviousVersion     string                 `json:"previousVersion"`
	PreviousChecksum    string                 `json:"previousChecksum"`
	Runtime             map[string]bool        `json:"runtime"`
}

type providerRegistration struct {
	Revision     int64             `json:"revision"`
	Wanted       bool              `json:"wanted"`
	Dependencies []string          `json:"dependencies"`
	Choices      map[string]string `json:"choices"`
}
