package cli

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	pprof "runtime/pprof"
	"runtime/trace"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "synheart",
	Short: "Synheart CLI - Mock HSI data generator for local development",
	Long: `Synheart CLI generates HSI-compatible sensor data streams
that mimic phone + wearable sources for local SDK development.

It eliminates dependency on physical devices during development,
providing repeatable scenarios for QA and demos.`,
	Example: strings.TrimSpace(`
synheart mock start
synheart mock list-scenarios
synheart receiver
synheart doctor
synheart version
`),
	SilenceUsage:  true,
	SilenceErrors: true,
}

var ui *UI
var profilingCleanup func()

// Execute runs the root command
func Execute() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	rootCmd.SetHelpTemplate(helpTemplate())
	rootCmd.SetUsageTemplate(usageTemplate())

	// Ensure profiling cleanup always runs, even if the command errors or exits early.
	defer func() {
		if profilingCleanup != nil {
			profilingCleanup()
			profilingCleanup = nil
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		// Ensure profiling cleanup runs on error/exit.
		if profilingCleanup != nil {
			profilingCleanup()
			profilingCleanup = nil
		}
		// At this point flags are parsed; UI is configured in init().
		if ui == nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		ui.Errorf("%v", err)
		ui.Printf("hint: run %s\n", ui.dim("synheart --help"))
		os.Exit(1)
	}
}

func init() {
	initRootFlags()
	rootCmd.AddCommand(mockCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(receiverCmd)
}

func initRootFlags() {
	rootCmd.PersistentFlags().StringVar(&globalOpts.Format, "format", "text", "Output format: text|json")
	rootCmd.PersistentFlags().BoolVar(&globalOpts.NoColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&globalOpts.Quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&globalOpts.Verbose, "verbose", "v", false, "Verbose logging")
	rootCmd.PersistentFlags().BoolVar(&globalOpts.Pprof, "pprof", false, "Enable pprof HTTP server")
	rootCmd.PersistentFlags().StringVar(&globalOpts.PprofAddr, "pprof-addr", globalOpts.PprofAddr, "Address for pprof HTTP server (host:port)")
	rootCmd.PersistentFlags().StringVar(&globalOpts.CPUProfile, "cpu-profile", "", "Write CPU profile to file")
	rootCmd.PersistentFlags().StringVar(&globalOpts.MemProfile, "mem-profile", "", "Write heap profile to file on exit")
	rootCmd.PersistentFlags().StringVar(&globalOpts.TraceProfile, "trace-profile", "", "Write execution trace to file")
	rootCmd.PersistentFlags().IntVar(&globalOpts.BlockProfileRate, "block-profile-rate", 0, "Enable block profiling with given rate (0 to disable)")
	rootCmd.PersistentFlags().IntVar(&globalOpts.MutexProfileFraction, "mutex-profile-fraction", 0, "Set mutex profile fraction (0 to disable)")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Cobra defaults to stdout/stderr; use the command writers so tests/redirects work.
		out := cmd.OutOrStdout()
		er := cmd.ErrOrStderr()
		ui = NewUI(out, er, globalOpts.NoColor, globalOpts.Quiet, globalOpts.Verbose)

		// Normalize format.
		globalOpts.Format = strings.ToLower(strings.TrimSpace(globalOpts.Format))
		if globalOpts.Format == "" {
			globalOpts.Format = "text"
		}
		if globalOpts.Format != "text" && globalOpts.Format != "json" {
			return fmt.Errorf("invalid --format %q (expected: text|json)", globalOpts.Format)
		}

		// Make sure help/usage can write to the right place.
		cmd.SetOut(out)
		cmd.SetErr(er)

		// Profiling and pprof setup.
		// Start pprof HTTP server if enabled.
		if globalOpts.Pprof {
			go func(addr string) {
				fmt.Fprintf(er, "pprof HTTP server listening on %s\n", addr)
				if err := http.ListenAndServe(addr, nil); err != nil {
					fmt.Fprintf(er, "pprof server error: %v\n", err)
				}
			}(globalOpts.PprofAddr)
		}

		// Configure runtime profiling knobs.
		if globalOpts.BlockProfileRate > 0 {
			runtime.SetBlockProfileRate(globalOpts.BlockProfileRate)
		}
		if globalOpts.MutexProfileFraction > 0 {
			runtime.SetMutexProfileFraction(globalOpts.MutexProfileFraction)
		}

		// Setup CPU / trace profiling if requested.
		var cpuFile *os.File
		var traceFile *os.File

		if globalOpts.CPUProfile != "" {
			f, err := os.Create(globalOpts.CPUProfile)
			if err != nil {
				return fmt.Errorf("failed to create cpu profile: %w", err)
			}
			cpuFile = f
			if err := pprof.StartCPUProfile(cpuFile); err != nil {
				_ = cpuFile.Close()
				return fmt.Errorf("failed to start cpu profiling: %w", err)
			}
			fmt.Fprintf(er, "CPU profiling started, writing to %s\n", globalOpts.CPUProfile)
		}

		if globalOpts.TraceProfile != "" {
			f, err := os.Create(globalOpts.TraceProfile)
			if err != nil {
				return fmt.Errorf("failed to create trace file: %w", err)
			}
			traceFile = f
			if err := trace.Start(traceFile); err != nil {
				_ = traceFile.Close()
				return fmt.Errorf("failed to start trace: %w", err)
			}
			fmt.Fprintf(er, "Execution trace started, writing to %s\n", globalOpts.TraceProfile)
		}

		// Register cleanup to run after command completes.
		profilingCleanup = func() {
			if cpuFile != nil {
				pprof.StopCPUProfile()
				_ = cpuFile.Close()
				fmt.Fprintf(er, "CPU profile written to %s\n", globalOpts.CPUProfile)
			}
			if globalOpts.MemProfile != "" {
				f, err := os.Create(globalOpts.MemProfile)
				if err == nil {
					runtime.GC()
					if err2 := pprof.Lookup("heap").WriteTo(f, 0); err2 != nil {
						fmt.Fprintf(er, "failed to write heap profile: %v\n", err2)
					} else {
						fmt.Fprintf(er, "Heap profile written to %s\n", globalOpts.MemProfile)
					}
					_ = f.Close()
				} else {
					fmt.Fprintf(er, "failed to create heap profile file: %v\n", err)
				}
			}
			if traceFile != nil {
				trace.Stop()
				_ = traceFile.Close()
				fmt.Fprintf(er, "Trace written to %s\n", globalOpts.TraceProfile)
			}
		}

		return nil
	}

	// Ensure profiling cleanup runs after every command.
	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		if profilingCleanup != nil {
			profilingCleanup()
			profilingCleanup = nil
		}
	}
}

func helpTemplate() string {
	// Minimal, modern-ish help output without extra dependencies.
	// Cobra will expand the template with command-specific values.
	return strings.TrimSpace(`
{{with (or .Long .Short)}}{{.}}{{end}}

Usage:
  {{.UseLine}}

{{if .HasAvailableSubCommands}}Commands:
{{range .Commands}}{{if (and (not .Hidden) .IsAvailableCommand)}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}
{{end}}

{{if .HasAvailableLocalFlags}}Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

{{if .HasAvailableInheritedFlags}}Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

{{if .HasExample}}Examples:
{{.Example}}{{end}}
`)
}

func usageTemplate() string {
	return strings.TrimSpace(`
Usage:
  {{.UseLine}}

Run 'synheart --help' for more information.
`)
}
