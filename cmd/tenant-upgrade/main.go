// Command tenant-upgrade validates mapping manifests and inventories frozen legacy SQL.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/elcomware/edupilot-os/platform/tenantupgrade"
)

type result struct {
	Version        int    `json:"version"`
	Schools        int    `json:"schools"`
	ManifestSHA256 string `json:"manifestSha256"`
}

const usage = "Usage: tenant-upgrade -mode=validate-manifest -manifest=PATH | -mode=inspect-placement | -mode=inventory-source -selection=PRIVATE_JSON | -mode=dry-run -manifest=PRIVATE_JSON -selection=PRIVATE_JSON"

func run(args []string, output, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("tenant-upgrade", flag.ContinueOnError)
	// The flag parser includes unrecognized input in its own messages.
	flags.SetOutput(io.Discard)
	mode := flags.String("mode", "", "operation: validate-manifest, inspect-placement, inventory-source or dry-run")
	path := flags.String("manifest", "", "path to the private JSON mapping manifest")
	selection := flags.String("selection", "", "path to the private source selection")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintln(diagnostics, usage)
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *mode == "inspect-placement" && *path == "" && *selection == "" && flags.NArg() == 0 {
		return inspectPlacement(output, diagnostics)
	}
	if *mode == "inventory-source" && *path == "" && *selection != "" && flags.NArg() == 0 {
		return inventorySource(*selection, output, diagnostics)
	}
	if *mode == "dry-run" && *path != "" && *selection != "" && flags.NArg() == 0 {
		return upgradeDryRun(*path, *selection, output, diagnostics)
	}
	if *mode != "validate-manifest" || *path == "" || *selection != "" || flags.NArg() != 0 {
		fmt.Fprintln(diagnostics, usage)
		return 2
	}
	file, err := os.Open(*path)
	if err != nil {
		fmt.Fprintln(diagnostics, "Cannot open manifest.")
		return 1
	}
	defer file.Close()
	manifest, err := tenantupgrade.ReadManifest(file)
	if err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	digest, err := manifest.Fingerprint()
	if err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	if err = json.NewEncoder(output).Encode(result{manifest.Version, len(manifest.Schools), digest}); err != nil {
		return 1
	}
	return 0
}
func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
