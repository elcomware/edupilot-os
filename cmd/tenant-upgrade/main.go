// Command tenant-upgrade validates mapping manifests and inventories frozen legacy SQL.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/elcomware/edupilot/platform/tenantupgrade"
)

type result struct {
	Version        int    `json:"version"`
	Schools        int    `json:"schools"`
	ManifestSHA256 string `json:"manifestSha256"`
}

const usage = "Usage: tenant-upgrade -mode=validate-manifest -manifest=PATH | -mode=inspect-placement | -mode=inventory-source -selection=PRIVATE_JSON | -mode=dry-run|rehearse-sql-fence -manifest=PRIVATE_JSON -selection=PRIVATE_JSON | -mode=verify-local-restore -manifest=PRIVATE_JSON -selection=PRIVATE_JSON -references=PRIVATE_JSON -root=RESTORED_ROOT"

func run(args []string, output, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("tenant-upgrade", flag.ContinueOnError)
	// The flag parser includes unrecognized input in its own messages.
	flags.SetOutput(io.Discard)
	mode := flags.String("mode", "", "operation: validate-manifest, inspect-placement, inventory-source, dry-run, rehearse-sql-fence or verify-local-restore")
	path := flags.String("manifest", "", "path to the private JSON mapping manifest")
	selection := flags.String("selection", "", "path to the private source selection")
	root := flags.String("root", "", "restored private document directory")
	references := flags.String("references", "", "preserved private object reference file")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintln(diagnostics, usage)
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	restore := *mode == "verify-local-restore"
	restoreOptions := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "root" || f.Name == "references" {
			restoreOptions = true
		}
	})
	if (restore && (*root == "" || *references == "")) || (!restore && restoreOptions) {
		fmt.Fprintln(diagnostics, usage)
		return 2
	}
	if *mode == "inspect-placement" && *path == "" && *selection == "" && flags.NArg() == 0 {
		return inspectPlacement(output, diagnostics)
	}
	if *mode == "inventory-source" && *path == "" && *selection != "" && flags.NArg() == 0 {
		return inventorySource(*selection, output, diagnostics)
	}
	if (*mode == "dry-run" || *mode == "rehearse-sql-fence" || restore) && *path != "" && *selection != "" && flags.NArg() == 0 {
		return checkUpgrade(*mode, *path, *selection, *references, *root, output, diagnostics)
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
