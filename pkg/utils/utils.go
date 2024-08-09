package utils

import (
	"fmt"
	"io"
)

func PrintVersion(o io.Writer, version string) {
	fmt.Fprintf(o, "jsonnet-debugger version %s\n", version)
}

func Usage(o io.Writer, version string) {
	PrintVersion(o, version)
	fmt.Fprintln(o)
	fmt.Fprintln(o, "jsonnice {<option>} { <filename> }")
	fmt.Fprintln(o)
	fmt.Fprintln(o, "Available options:")
	fmt.Fprintln(o, "  -h / --help                This message")
	fmt.Fprintln(o, "  -e / --exec                Treat filename as code")
	fmt.Fprintln(o, "  -J / --jpath <dir>         Specify an additional library search dir")
	fmt.Fprintln(o, "  -d / --dap                 Start a debug-adapter-protocol server")
	fmt.Fprintln(o, "  -s / --stdin               Start a debug-adapter-protocol session using stdion/stdout for communication")
	fmt.Fprintln(o, "  -l / --log-level           Set the log level. Allowed values: debug,info,warn,error")
	fmt.Fprintln(o, "  --tlaCode				  Set TLA")
	fmt.Fprintln(o, "  --version                  Print version")
	fmt.Fprintln(o)
	fmt.Fprintln(o, "In all cases:")
	fmt.Fprintln(o, "  Multichar options are expanded e.g. -abc becomes -a -b -c.")
	fmt.Fprintln(o, "  The -- option suppresses option processing for subsequent arguments.")
	fmt.Fprintln(o, "  Note that since filenames and jsonnet programs can begin with -, it is")
	fmt.Fprintln(o, "  advised to use -- if the argument is unknown, e.g. jsonnice -- \"$FILENAME\".")
}
