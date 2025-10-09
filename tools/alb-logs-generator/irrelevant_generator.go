package main

// IrrelevantGenerator is a placeholder for producing irrelevant fields.
type IrrelevantGenerator struct{}

// NewIrrelevantGenerator constructs a new irrelevant generator.
func NewIrrelevantGenerator() *IrrelevantGenerator {
	return &IrrelevantGenerator{}
}

// Generate returns zero-valued irrelevant fields.
func (g *IrrelevantGenerator) Generate() (IrrelevantFields, error) {
	return IrrelevantFields{}, nil
}
