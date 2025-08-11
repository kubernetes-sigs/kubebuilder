package scaffolds

// FeatureGateScaffolder interface for scaffolders that support feature gates
type FeatureGateScaffolder interface {
	SetWithFeatureGates(bool)
}
