package agent

// DefaultVersion is the default -version flag when launching the binary.
// Health reports using this value must not overwrite a version already stored on the platform
// (e.g. set during bootstrap) so the UI-declared version remains visible when the process still uses defaults.
const DefaultVersion = "v0.1.0"
