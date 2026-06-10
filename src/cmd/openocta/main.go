package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/openocta/openocta/pkg/backup"
	"github.com/openocta/openocta/pkg/paths"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "openocta",
		Short: "OpenOcta gateway and operations CLI",
	}
	root.AddCommand(newBackupCmd(), newRestoreCmd(), newVerifyCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func stateDirFlag(cmd *cobra.Command) string {
	dir, _ := cmd.Flags().GetString("state-dir")
	if dir != "" {
		return dir
	}
	return paths.ResolveStateDir(os.Getenv)
}

func newBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create an online backup of openocta.db and attachment directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			output, _ := cmd.Flags().GetString("output")
			if output == "" {
				return fmt.Errorf("--output is required")
			}
			manifest, err := backup.Create(backup.Options{
				StateDir:   stateDirFlag(cmd),
				OutputPath: output,
			})
			if err != nil {
				return err
			}
			b, _ := json.MarshalIndent(manifest, "", "  ")
			fmt.Fprintf(os.Stdout, "backup written to %s\n%s\n", output, string(b))
			return nil
		},
	}
	cmd.Flags().String("state-dir", "", "State directory (default: OPENOCTA_STATE_DIR or ~/.openocta)")
	cmd.Flags().StringP("output", "o", "", "Output .tar.gz path")
	return cmd
}

func newRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore openocta.db and attachments from a backup archive",
		RunE: func(cmd *cobra.Command, args []string) error {
			input, _ := cmd.Flags().GetString("input")
			if input == "" {
				return fmt.Errorf("--input is required")
			}
			force, _ := cmd.Flags().GetBool("force")
			manifest, err := backup.Restore(backup.RestoreOptions{
				ArchivePath: input,
				StateDir:    stateDirFlag(cmd),
				Force:       force,
			})
			if err != nil {
				return err
			}
			b, _ := json.MarshalIndent(manifest, "", "  ")
			fmt.Fprintf(os.Stdout, "restore completed into %s\n%s\n", stateDirFlag(cmd), string(b))
			return nil
		},
	}
	cmd.Flags().String("state-dir", "", "State directory (default: OPENOCTA_STATE_DIR or ~/.openocta)")
	cmd.Flags().StringP("input", "i", "", "Backup .tar.gz path")
	cmd.Flags().Bool("force", false, "Overwrite existing state directory contents")
	return cmd
}

func newVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup-verify",
		Short: "Verify backup archive checksums without restoring",
		RunE: func(cmd *cobra.Command, args []string) error {
			input, _ := cmd.Flags().GetString("input")
			if input == "" {
				return fmt.Errorf("--input is required")
			}
			manifest, err := backup.VerifyArchive(input)
			if err != nil {
				return err
			}
			b, _ := json.MarshalIndent(manifest, "", "  ")
			fmt.Fprintf(os.Stdout, "backup verified: %s\n%s\n", input, string(b))
			return nil
		},
	}
	cmd.Flags().StringP("input", "i", "", "Backup .tar.gz path")
	return cmd
}
