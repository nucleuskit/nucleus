package capability

import (
	"fmt"
	"strings"

	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/capcatalog"
)

func genericComponentTemplate(spec capcatalog.Spec, provider string) string {
	typeName := goExportName(spec.Name) + "Component"
	configName := goExportName(spec.Name) + "Config"
	return fmt.Sprintf(`package %s

import (
	"context"
	"fmt"
	"os"
)

const (
	// CapabilityName is the manifest capability represented by this component.
	CapabilityName = %q
	// ProviderName is the manifest provider represented by this component.
	ProviderName = %q
	// DSNEnv is the optional environment variable used by this provider.
	DSNEnv = %q
)

// %s carries service-owned configuration for the %s capability.
type %s struct {
	Provider string `+"`json:\"provider\" yaml:\"provider\"`"+`
	DSNEnv   string `+"`json:\"dsn_env,omitempty\" yaml:\"dsn_env,omitempty\"`"+`
	DSN      string `+"`json:\"-\" yaml:\"-\"`"+`
}

// DefaultConfig returns the generated default provider metadata.
func DefaultConfig() %s {
	return %s{
		Provider: ProviderName,
		DSNEnv:   DSNEnv,
	}
}

// LoadConfigFromEnv loads non-secret provider configuration from the process environment.
func LoadConfigFromEnv() %s {
	cfg := DefaultConfig()
	if cfg.DSNEnv != "" {
		cfg.DSN = os.Getenv(cfg.DSNEnv)
	}
	return cfg
}

// Validate checks static configuration without opening network connections.
func (c %s) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("%s provider is required")
	}
	return nil
}

// %s is a compile-safe service-side capability adapter.
type %s struct {
	config %s
}

// New creates the capability adapter without importing provider SDKs.
func New(config %s) (*%s, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &%s{config: config}, nil
}

// NewFromEnv creates the capability adapter from environment-backed configuration.
func NewFromEnv() (*%s, error) {
	return New(LoadConfigFromEnv())
}

// Config returns the effective capability configuration.
func (c *%s) Config() %s {
	if c == nil {
		return DefaultConfig()
	}
	return c.config
}

// Ready validates local configuration and honors context cancellation.
func (c *%s) Ready(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.Config().Validate()
}

// Shutdown releases provider resources. The generated adapter owns no resources.
func (c *%s) Shutdown(ctx context.Context) error {
	return ctx.Err()
}
`, goPackageName(spec.Name), spec.Name, provider, spec.DSNEnv,
		configName, spec.Name, configName,
		configName, configName,
		configName,
		configName, spec.Name,
		typeName, typeName, configName,
		configName, typeName,
		typeName,
		typeName,
		typeName, configName,
		typeName,
		typeName)
}

func genericAppTemplate(module string, spec capcatalog.Spec, provider string) string {
	typeName := goExportName(spec.Name) + "Capability"
	componentAlias := "component" + goExportName(spec.Name)
	return fmt.Sprintf(`package app

import (
	"context"

	%s "%s/internal/component/%s"
)

const (
	// %sName is the manifest capability name.
	%sName = %q
	// %sProvider is the manifest provider name.
	%sProvider = %q
)

// %s owns the generated %s capability adapter.
type %s struct {
	Component *%s.%s
}

// New%s creates the generated capability adapter from environment-backed configuration.
func New%s() (*%s, error) {
	component, err := %s.NewFromEnv()
	if err != nil {
		return nil, err
	}
	return &%s{Component: component}, nil
}

// Shutdown releases generated capability resources.
func (c *%s) Shutdown(ctx context.Context) error {
	if c == nil || c.Component == nil {
		return nil
	}
	return c.Component.Shutdown(ctx)
}
`, componentAlias, module, goPackageName(spec.Name),
		typeName, typeName, spec.Name,
		typeName, typeName, provider,
		typeName, spec.Name, typeName, componentAlias, goExportName(spec.Name)+"Component",
		typeName, typeName, typeName,
		componentAlias,
		typeName,
		typeName)
}

func genericDocsTemplate(spec capcatalog.Spec, provider string) string {
	dsnLine := ""
	if spec.DSNEnv != "" {
		dsnLine = "- Runtime DSN is read from " + spec.DSNEnv + " and must not be committed.\n"
	}
	return fmt.Sprintf("# Capability: %s/%s\n\n"+
		"- Capability %q is declared in nucleus.yaml.\n"+
		"- Provider %q is recorded as explicit manifest metadata.\n"+
		"%s- Generated service code is compile-safe and owns no hidden global provider state.\n"+
		"- Replace the generated component with concrete bridge wiring when the selected provider package exists.\n",
		spec.Name, provider, spec.Name, provider, dsnLine)
}

func postgresComponentTemplate() string {
	return `package sql

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

const (
	// CapabilityName is the manifest capability represented by this component.
	CapabilityName = "sql"
	// ProviderName is the manifest provider represented by this component.
	ProviderName = "postgres"
	// DatabaseDSNEnv is the environment variable used for the PostgreSQL DSN.
	DatabaseDSNEnv = "NUCLEUS_DATABASE_DSN"
)

// PostgresConfig carries service-owned PostgreSQL configuration.
type PostgresConfig struct {
	Provider     string ` + "`json:\"provider\" yaml:\"provider\"`" + `
	Driver       string ` + "`json:\"driver\" yaml:\"driver\"`" + `
	DSNEnv       string ` + "`json:\"dsn_env\" yaml:\"dsn_env\"`" + `
	DSN          string ` + "`json:\"-\" yaml:\"-\"`" + `
	MaxOpenConns int    ` + "`json:\"max_open_conns\" yaml:\"max_open_conns\"`" + `
	MaxIdleConns int    ` + "`json:\"max_idle_conns\" yaml:\"max_idle_conns\"`" + `
}

// DefaultPostgresConfig returns conservative database/sql pool defaults.
func DefaultPostgresConfig() PostgresConfig {
	return PostgresConfig{
		Provider:     ProviderName,
		Driver:       "postgres",
		DSNEnv:       DatabaseDSNEnv,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	}
}

// LoadPostgresConfigFromEnv loads secret DSN material from the process environment.
func LoadPostgresConfigFromEnv() PostgresConfig {
	cfg := DefaultPostgresConfig()
	cfg.DSN = os.Getenv(cfg.DSNEnv)
	return cfg
}

// Validate checks static database configuration without opening a connection.
func (c PostgresConfig) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("database provider is required")
	}
	if c.Driver == "" {
		return fmt.Errorf("database driver is required")
	}
	if c.DSNEnv == "" {
		return fmt.Errorf("database DSN environment name is required")
	}
	return nil
}

// NewPostgresDB opens a PostgreSQL database handle when a DSN is configured.
func NewPostgresDB(config PostgresConfig) (*sql.DB, func(context.Context) error, error) {
	if err := config.Validate(); err != nil {
		return nil, nil, err
	}
	if config.DSN == "" {
		return nil, func(context.Context) error { return nil }, nil
	}
	db, err := sql.Open(config.Driver, config.DSN)
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	cleanup := func(context.Context) error {
		return db.Close()
	}
	return db, cleanup, nil
}

// NewPostgresDBFromEnv opens a PostgreSQL database handle from environment configuration.
func NewPostgresDBFromEnv() (*sql.DB, func(context.Context) error, error) {
	return NewPostgresDB(LoadPostgresConfigFromEnv())
}
`
}

func postgresAppTemplate(module string) string {
	return fmt.Sprintf(`package app

import (
	"context"
	"database/sql"

	componentsql "%s/internal/component/sql"
)

const (
	// SQLCapabilityName is the manifest capability name.
	SQLCapabilityName = "sql"
	// SQLCapabilityProvider is the manifest provider name.
	SQLCapabilityProvider = "postgres"
)

// SQLCapability owns the generated PostgreSQL database handle.
type SQLCapability struct {
	DB      *sql.DB
	cleanup func(context.Context) error
}

// NewSQLCapability creates the generated PostgreSQL database handle from environment configuration.
func NewSQLCapability() (*SQLCapability, error) {
	db, cleanup, err := componentsql.NewPostgresDBFromEnv()
	if err != nil {
		return nil, err
	}
	return &SQLCapability{DB: db, cleanup: cleanup}, nil
}

// Shutdown closes the generated database handle when it was opened.
func (c *SQLCapability) Shutdown(ctx context.Context) error {
	if c == nil || c.cleanup == nil {
		return nil
	}
	return c.cleanup(ctx)
}
`, module)
}

func postgresRepositoryTemplate() string {
	return `package postgres

import (
	"context"
	"database/sql"
)

// Repository is a service-owned PostgreSQL repository base.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a PostgreSQL repository base from an optional database handle.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// DB returns the underlying database handle for service-owned query code.
func (r *Repository) DB() *sql.DB {
	if r == nil {
		return nil
	}
	return r.db
}

// Ping verifies the database handle when it is configured.
func (r *Repository) Ping(ctx context.Context) error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.PingContext(ctx)
}
`
}

func postgresMigrationTemplate() string {
	return `-- Generated by nucleus capability add sql --provider postgres.
-- Add service-owned tables in a follow-up migration before applying it.
`
}

func postgresDocsTemplate(spec capcatalog.Spec, provider string) string {
	return fmt.Sprintf("# Capability: %s/%s\n\n"+
		"- Capability %q is declared in nucleus.yaml.\n"+
		"- Provider %q is recorded as explicit manifest metadata.\n"+
		"- The generated component uses database/sql and the PostgreSQL driver through an isolated service-side package.\n"+
		"- Runtime DSN is read from NUCLEUS_DATABASE_DSN and must not be committed.\n"+
		"- The generated database handle is optional: empty DSN keeps local tests and generated services runnable.\n",
		spec.Name, provider, spec.Name, provider)
}

func providerFileName(provider string) string {
	if provider == "" {
		return "noop"
	}
	return strings.NewReplacer("/", "_", "-", "_").Replace(provider)
}

func goPackageName(value string) string {
	value = strings.ToLower(value)
	var builder strings.Builder
	for _, ch := range value {
		if ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
			builder.WriteRune(ch)
		}
	}
	if builder.Len() == 0 {
		return "capability"
	}
	return builder.String()
}

func goExportName(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == '_' || r == '/' || r == '.'
	})
	if len(parts) == 1 {
		if initialism, ok := initialismName(strings.ToLower(parts[0])); ok {
			return initialism
		}
	}
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if initialism, ok := initialismName(strings.ToLower(part)); ok {
			builder.WriteString(initialism)
			continue
		}
		for index, ch := range part {
			if index == 0 && ch >= 'a' && ch <= 'z' {
				ch = ch - 'a' + 'A'
			}
			if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' {
				builder.WriteRune(ch)
			}
		}
	}
	if builder.Len() == 0 {
		return "Capability"
	}
	return builder.String()
}

func initialismName(value string) (string, bool) {
	switch value {
	case "http":
		return "HTTP", true
	case "httpclient":
		return "HTTPClient", true
	case "grpc":
		return "GRPC", true
	case "sql":
		return "SQL", true
	case "dsn":
		return "DSN", true
	case "kv":
		return "KV", true
	case "mq":
		return "MQ", true
	case "redis":
		return "Redis", true
	case "mongo":
		return "Mongo", true
	case "errortracker":
		return "ErrorTracker", true
	default:
		return "", false
	}
}
