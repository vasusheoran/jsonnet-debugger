package main

import (
	"encoding/json"
	"fmt"
	"github.com/grafana/jsonnet-debugger/pkg/utils"
	"io"
	"log/slog"
	"os"
	"path"
)

var (
	// Set with `-ldflags="-X 'main.version=<version>'"`
	Version = "0.0.2"
)

var logger *utils.CustomLogger

func main() {
	cfg := config{
		jpath:    []string{},
		extCode:  make(map[string]interface{}),
		tlaCode:  make(map[string]interface{}),
		logLevel: slog.LevelDebug,
	}
	file, err := os.OpenFile("dap.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	logger = utils.NewCustomLogger(file, "")

	status, err := processArgs(os.Args[1:], &cfg, os.Stdout)

	if err != nil {
		logger.Error(err.Error())
	}

	switch status {
	case processArgsStatusContinue:
		break
	case processArgsStatusSuccessUsage:
		utils.Usage(file, Version)
		os.Exit(0)
	case processArgsStatusFailureUsage:
		if err != nil {
			logger.Error(err.Error())
		}
		utils.Usage(file, Version)
		os.Exit(1)
	case processArgsStatusSuccess:
		os.Exit(0)
	case processArgsStatusFailure:
		os.Exit(1)
	}

	if cfg.dap {
		var err error
		if cfg.stdin {
			err = dapStdin(cfg)
		} else {
			err = dapServer("54321", cfg)
		}
		if err != nil {
			logger.Error("dap server terminated", "err", err)
		}
		return
	}

	inputFile := cfg.inputFile
	input := safeReadInput(cfg.filenameIsCode, &inputFile)
	if !cfg.filenameIsCode {
		cfg.jpath = append(cfg.jpath, path.Dir(inputFile))
	}
	repl := MakeReplDebugger(inputFile, input, cfg.jpath)
	repl.Run()
}

type config struct {
	inputFile      string
	filenameIsCode bool
	dap            bool
	jpath          []string
	logLevel       slog.Level
	stdin          bool
	extCode        map[string]interface{}
	tlaCode        map[string]interface{}
}

type processArgsStatus int

const (
	processArgsStatusContinue     = iota
	processArgsStatusSuccessUsage = iota
	processArgsStatusFailureUsage = iota
	processArgsStatusSuccess      = iota
	processArgsStatusFailure      = iota
)

// nextArg retrieves the next argument from the commandline.
func nextArg(i *int, args []string) string {
	(*i)++
	if (*i) >= len(args) {
		fmt.Fprintln(os.Stderr, "Expected another commandline argument.")
		os.Exit(1)
	}
	return args[*i]
}

// simplifyArgs transforms an array of commandline arguments so that
// any -abc arg before the first -- (if any) are expanded into
// -a -b -c.
func simplifyArgs(args []string) (r []string) {
	r = make([]string, 0, len(args)*2)
	for i, arg := range args {
		if arg == "--" {
			for j := i; j < len(args); j++ {
				r = append(r, args[j])
			}
			break
		}
		if len(arg) > 2 && arg[0] == '-' && arg[1] != '-' {
			for j := 1; j < len(arg); j++ {
				r = append(r, "-"+string(arg[j]))
			}
		} else {
			r = append(r, arg)
		}
	}
	return
}

func processArgs(givenArgs []string, cfg *config, file *os.File) (processArgsStatus, error) {
	args := simplifyArgs(givenArgs)

	remainingArgs := make([]string, 0, len(args))
	i := 0

	for ; i < len(args); i++ {
		arg := args[i]
		if arg == "-h" || arg == "--help" {
			return processArgsStatusSuccessUsage, nil
		} else if arg == "-v" || arg == "--version" {
			utils.PrintVersion(file, Version)
			return processArgsStatusSuccess, nil
		} else if arg == "-e" || arg == "--exec" {
			cfg.filenameIsCode = true
		} else if arg == "-s" || arg == "--stdin" {
			cfg.stdin = true
		} else if arg == "--" {
			// All subsequent args are not options.
			i++
			for ; i < len(args); i++ {
				remainingArgs = append(remainingArgs, args[i])
			}
			break
		} else if arg == "-J" || arg == "--jpath" {
			dir := nextArg(&i, args)
			if len(dir) == 0 {
				return processArgsStatusFailure, fmt.Errorf("-J argument was empty string")
			}
			cfg.jpath = append(cfg.jpath, dir)
		} else if arg == "-d" || arg == "--dap" {
			cfg.dap = true
		} else if arg == "-l" || arg == "--log-level" {
			level := nextArg(&i, args)
			if len(level) == 0 {
				return processArgsStatusFailure, fmt.Errorf("no log level specified")
			}
			slvl := slog.LevelDebug
			switch level {
			case "debug":
				slvl = slog.LevelDebug
			case "info":
				slvl = slog.LevelInfo
			case "error":
				slvl = slog.LevelError
			default:
				return processArgsStatusFailure, fmt.Errorf("invalid log level %s. Allowed: debug,info,warn,error", level)
			}
			cfg.logLevel = slvl
		} else if arg == "--extCode" {
			argValue := nextArg(&i, args)
			_, err := parseCode(cfg.extCode, argValue)
			if err != nil {
				logger.Error("err", err)
				return processArgsStatusFailure, err
			}
		} else if arg == "--tlaCode" {
			argValue := nextArg(&i, args)
			_, err := parseCode(cfg.tlaCode, argValue)
			if err != nil {
				logger.Error("err", err)
				return processArgsStatusFailure, err
			}
		} else if len(arg) > 1 && arg[0] == '-' {
			return processArgsStatusFailure, fmt.Errorf("unrecognized argument: %s", arg)
		} else {
			remainingArgs = append(remainingArgs, arg)
		}
	}

	if cfg.dap {
		return processArgsStatusContinue, nil
	}

	want := "filename"
	if cfg.filenameIsCode {
		want = "code"
	}
	if len(remainingArgs) == 0 {
		return processArgsStatusFailureUsage, fmt.Errorf("must give %s", want)
	}
	if len(remainingArgs) != 1 {
		// Should already have been caught by processArgs.
		panic("Internal error: expected a single input file.")
	}

	cfg.inputFile = remainingArgs[0]
	return processArgsStatusContinue, nil
}

func parseCode(codeMap map[string]interface{}, unparsed string) (map[string]interface{}, error) {
	var code map[string]interface{}
	err := json.Unmarshal([]byte(unparsed), &code)
	if err != nil {
		panic(err)
	}

	if codeMap == nil {
		codeMap = make(map[string]interface{}, len(code))
	}

	for key, value := range code {
		codeMap[key] = value
	}

	return codeMap, nil
}

// readInput gets Jsonnet code from the given place (file, commandline, stdin).
// It also updates the given filename to <stdin> or <cmdline> if it wasn't a
// real filename.
func readInput(filenameIsCode bool, filename *string) (input string, err error) {
	if filenameIsCode {
		input, err = *filename, nil
		*filename = "<cmdline>"
	} else if *filename == "-" {
		var bytes []byte
		bytes, err = io.ReadAll(os.Stdin)
		input = string(bytes)
		*filename = "<stdin>"
	} else {
		var bytes []byte
		bytes, err = os.ReadFile(*filename)
		input = string(bytes)
	}
	return
}

// safeReadInput runs ReadInput, exiting the process if there was a problem.
func safeReadInput(filenameIsCode bool, filename *string) string {
	output, err := readInput(filenameIsCode, filename)
	if err != nil {
		var op string
		switch typedErr := err.(type) {
		case *os.PathError:
			op = typedErr.Op
			err = typedErr.Err
		}
		if op == "open" {
			fmt.Fprintf(os.Stderr, "Opening input file: %s: %s\n", *filename, err.Error())
		} else if op == "read" {
			fmt.Fprintf(os.Stderr, "Reading input file: %s: %s\n", *filename, err.Error())
		} else {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		os.Exit(1)
	}
	return output
}
