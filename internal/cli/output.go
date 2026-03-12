package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var outputFormat string

func printFormatted(data any) error {
	switch outputFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case "yaml":
		return yaml.NewEncoder(os.Stdout).Encode(data)
	default:
		return fmt.Errorf("unsupported output format %q", outputFormat)
	}
}

func isStructuredOutput() bool {
	return outputFormat == "json" || outputFormat == "yaml"
}
