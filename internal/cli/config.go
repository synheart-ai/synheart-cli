package cli

// GlobalOptions are shared flags that apply across commands.
type GlobalOptions struct {
	Format  string
	NoColor bool
	Quiet   bool
	Verbose bool

	// Profiling options
	Pprof                bool   // enable HTTP pprof endpoints
	PprofAddr            string // address for pprof server (host:port)
	CPUProfile           string // path to write CPU profile
	MemProfile           string // path to write heap profile
	TraceProfile         string // path to write execution trace
	BlockProfileRate     int    // runtime.SetBlockProfileRate
	MutexProfileFraction int    // runtime.SetMutexProfileFraction
}

var globalOpts = GlobalOptions{
	Format:    "text",
	PprofAddr: "127.0.0.1:6060",
}
