package main

// RelevantGenerator is a placeholder for producing relevant fields.
type RelevantGenerator struct{}

// NewRelevantGenerator constructs a new relevant generator.
func NewRelevantGenerator() *RelevantGenerator {
	return &RelevantGenerator{}
}

// Generate returns zero-valued relevant fields.
func (g *RelevantGenerator) Generate(input RelevantInput) (RelevantFields, error) {
	return RelevantFields{}, nil
}
