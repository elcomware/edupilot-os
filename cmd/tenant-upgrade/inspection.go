package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elcomware/edupilot-os/modules"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/tenantupgrade"
)

func nativeInspectionSettings() (settings database.Settings, err error) {
	// Native configuration validation can panic before the framework's provider
	// would exit successfully on bad .env/APP_KEY. Do not print its private input.
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("native configuration unavailable")
		}
	}()
	return database.ConfiguredSettings()
}
func inspectPlacement(output, diagnostics io.Writer) int {
	settings, err := nativeInspectionSettings()
	if err != nil {
		fmt.Fprintln(diagnostics, "Cannot read native database configuration.")
		return 1
	}
	keys := []string{}
	for _, definition := range modules.Definitions() {
		if definition.Migration != nil {
			keys = append(keys, definition.Key)
		}
	}
	report := tenantupgrade.InspectPlacement(context.Background(), settings, keys)
	if err := json.NewEncoder(output).Encode(report); err != nil {
		return 1
	}
	if !report.Complete || len(report.Issues) > 0 {
		return 1
	}
	return 0
}
