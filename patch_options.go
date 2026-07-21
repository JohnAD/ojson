package ojson

// PatchOption configures patch apply, validate, and diff behavior.
type PatchOption interface {
	applyPatchOption(*patchConfig)
}

type patchConfig struct {
	schema    *JSONSchema
	hasSchema bool
}

func newPatchConfig(doc JSONValue, opts ...PatchOption) patchConfig {
	cfg := patchConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt.applyPatchOption(&cfg)
		}
	}
	if !cfg.hasSchema {
		if attached := doc.Schema(); attached != nil {
			cfg.schema = attached
			cfg.hasSchema = true
		}
	}
	return cfg
}

type withPatchSchemaOption struct {
	schema JSONSchema
}

// WithPatchSchema supplies an explicit schema for patch operations.
// When present it takes precedence over a schema attached to the document.
func WithPatchSchema(schema JSONSchema) PatchOption {
	return withPatchSchemaOption{schema: schema}
}

func (o withPatchSchemaOption) applyPatchOption(cfg *patchConfig) {
	cfg.schema = &o.schema
	cfg.hasSchema = true
}
