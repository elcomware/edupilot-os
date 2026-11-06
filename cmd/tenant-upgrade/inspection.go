package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

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

func inventorySource(selectionPath string, output, diagnostics io.Writer) int {
	info, err := os.Lstat(selectionPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		fmt.Fprintln(diagnostics, "Cannot read a private regular source selection.")
		return 1
	}
	file, err := os.Open(selectionPath)
	if err != nil {
		fmt.Fprintln(diagnostics, "Cannot read a private regular source selection.")
		return 1
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		fmt.Fprintln(diagnostics, "Cannot read a private regular source selection.")
		return 1
	}
	selection, err := tenantupgrade.ReadSourceSelection(file)
	if err != nil {
		fmt.Fprintln(diagnostics, "Invalid private source selection.")
		return 1
	}
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
	report := tenantupgrade.InventorySource(context.Background(), settings, keys, selection)
	if err := json.NewEncoder(output).Encode(report); err != nil {
		return 1
	}
	if !report.Complete || len(report.Issues) > 0 {
		return 1
	}
	return 0
}

func upgradeDryRun(manifestPath, selectionPath string, output, diagnostics io.Writer) int {
	manifestFile, err := openPrivateRegular(manifestPath)
	if err != nil {
		fmt.Fprintln(diagnostics, "Cannot read a private regular mapping manifest.")
		return 1
	}
	defer manifestFile.Close()
	manifest, err := tenantupgrade.ReadManifest(manifestFile)
	if err != nil {
		fmt.Fprintln(diagnostics, "Invalid private mapping manifest.")
		return 1
	}
	selectionFile, err := openPrivateRegular(selectionPath)
	if err != nil {
		fmt.Fprintln(diagnostics, "Cannot read a private regular source selection.")
		return 1
	}
	defer selectionFile.Close()
	selection, err := tenantupgrade.ReadSourceSelection(selectionFile)
	if err != nil {
		fmt.Fprintln(diagnostics, "Invalid private source selection.")
		return 1
	}
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
	report, err := tenantupgrade.NewUpgrade(settings, keys, manifest, selection).DryRun(context.Background())
	if err != nil {
		fmt.Fprintln(diagnostics, "Upgrade dry-run did not complete.")
		return 1
	}
	if err := json.NewEncoder(output).Encode(report); err != nil {
		return 1
	}
	if !report.Complete || !report.ReadyForPrepare || len(report.Issues) != 0 {
		return 1
	}
	return 0
}

func openPrivateRegular(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("private regular file unavailable")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("private regular file unavailable")
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		file.Close()
		return nil, fmt.Errorf("private regular file unavailable")
	}
	return file, nil
}
