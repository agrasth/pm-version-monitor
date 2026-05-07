package cmd

import (
	"fmt"
	"os"

	"github.com/jfrog/pm-version-monitor/internal/generator"
	"github.com/jfrog/pm-version-monitor/internal/state"
)

// RunMatrix generates the HTML compatibility matrix from state and writes it to outputPath.
func RunMatrix(statePath, outputPath string) error {
	s, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file %s: %w", outputPath, err)
	}
	defer f.Close()

	if err := generator.Generate(s, f); err != nil {
		return fmt.Errorf("generating matrix: %w", err)
	}

	fmt.Fprintf(os.Stdout, "[pm-monitor] matrix written to %s\n", outputPath)
	return nil
}
