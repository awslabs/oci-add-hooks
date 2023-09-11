package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const (
	// Size of the buffer for catching os.Signal sent to this process
	signalBufferSize = 32

	// For the exit code, I only added createLogFailure, 
	// I'm not quite sure what numbers are included in runc's ExitStatus, 
	// I chose 111 for createLogFailure, hopefully it won't overlap with runc's ExitStatus
	exitCodeFailure  = 1
	createLogFailure = 111
)

var (
	errUnableToFindRuntime = errors.New("unable to find runtime")

	commit string

	logMode bool = false
)

func main() {
	// We are doing manual flag parsing b/c the default flag package
	// doesn't have the ability to parse only some flags and ignore unknown
	// ones. Just requiring positional arguments for simplicity.
	// We are expecting command line like one of the following:
	// self --version
	// self --hook-config-path /path/to/hookcfg --runtime-path /path/to/runc, ... runtime flags
	// If we don't match one of these these, we can exit
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println("commit:", commit)
		os.Exit(0)
	} else if len(os.Args) < 6 || (os.Args[1] != "--hook-config-path" && os.Args[3] != "--runtime-path") {
		os.Exit(exitCodeFailure)
	}
	// If are args are present, grab the values
	hookConfigPath := os.Args[2]
	runcPath := os.Args[4]
	passthroughArgs := os.Args[5:]
	// Check if --log-path flag is provided
	if len(passthroughArgs) > 0 && passthroughArgs[0] == "--log-path" {
		if len(passthroughArgs) < 2 {
			os.Exit(exitCodeFailure)
		}
		logPath := passthroughArgs[1]
		passthroughArgs = passthroughArgs[2:]
		if createLogFile(logPath) != 0 {
			os.Exit(createLogFailure)
		}
		logMode = true
	}
	if logMode {
		log.Println("Running oci-add-hooks")
		log.Println("Oci-add-hooks arguments right")
	}
	os.Exit(run(hookConfigPath, runcPath, passthroughArgs))
}

func run(hookConfigPath, runcPath string, runcArgs []string) int {
	// If required args aren't present, bail
	if hookConfigPath == "" || runcPath == "" {
		if logMode {
			log.Println("Error: hookConfigPath or runcPath is \"\"")
		}
		return exitCodeFailure
	}
	if logMode {
		log.Printf("HookconfigPath: %v\n", hookConfigPath)
		log.Printf("RuncPath: %v\n", runcPath)
		log.Printf("RuncArgs: \n%v\n", runcArgs)
	}
	// If a hookConfigPath passed, process the bundle and pass modified
	// spec to runc
	return processBundle(hookConfigPath, runcPath, runcArgs)
}

func processBundle(hookPath, runcPath string, runcArgs []string) int {
	// find the bundle json location
	for i, val := range runcArgs {
		if val == "--bundle" && i != len(runcArgs)-1 {
			// get the bundle Path
			bundlePath := runcArgs[i+1]
			bundlePath = filepath.Join(bundlePath, "config.json")
			// Add the hooks from hookPath to our bundle/config.json
			if logMode {
				log.Printf("BundleFile: \n%v\n", bundlePath)
			}
			merged, err := addHooks(bundlePath, hookPath)
			if err != nil {
				if logMode {
					log.Printf("Error: %v\n", err)
				}
				return exitCodeFailure
			}
			err = merged.writeFile(bundlePath)
			if err != nil {
				if logMode {
					log.Printf("Error: %v\n", err)
				}
				return exitCodeFailure
			}
			if logMode {
				log.Println("Add hooks to bundlefile success")
			}
			break
		}
	}
	// launch runc
	path, err := verifyRuntimePath(runcPath)
	if err != nil {
		if logMode {
			log.Printf("Error: runc path is wrong: %v\n", err)
		}
		return exitCodeFailure
	}
	return launchRunc(path, runcArgs)
}

func verifyRuntimePath(userDefinedRuncPath string) (string, error) {
	info, err := os.Stat(userDefinedRuncPath)
	if err == nil && !info.Mode().IsDir() && info.Mode().IsRegular() {
		return userDefinedRuncPath, nil
	}
	return "", errUnableToFindRuntime
}

// Launch runc with the provided args
func launchRunc(runcPath string, runcArgs []string) int {
	cmd := prepareCommand(runcPath, runcArgs)
	proc := make(chan os.Signal, signalBufferSize)
	// Handle signals before we start command to make sure we don't
	// miss any related to cmd.
	signal.Notify(proc)
	err := cmd.Start()
	if err != nil {
		if logMode {
			log.Printf("Error: runc start failed: %v\n", err)
		}
		return exitCodeFailure
	}
	if logMode {
		log.Println("Running runc")
	}
	// Forward signals after we start command
	go func() {
		for sig := range proc {
			cmd.Process.Signal(sig)
		}
	}()

	err = cmd.Wait()

	return processRuncError(err)
}

func processRuncError(err error) int {
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			// We had a nonzero exitCode
			if code, ok := exit.Sys().(syscall.WaitStatus); ok {
				// and the code is retrievable
				// so we exit with the same code
				return code.ExitStatus()
			}
		}
		// If we can't get the error code, still exit with error
		return exitCodeFailure
	}
	return 0
}

func prepareCommand(runcPath string, args []string) *exec.Cmd {
	cmd := exec.Command(runcPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// Add hooks specified inside hookPath to the bundle specified in args
func addHooks(bundlePath, hookPath string) (*config, error) {
	specHooks, err := readHooks(bundlePath)
	if err != nil {
		if logMode {
			log.Println("Error: read bundlePath hooks failed")
		}
		return nil, err
	}
	addHooks, err := readHooks(hookPath)
	if err != nil {
		if logMode {
			log.Println("Error: read hookPath hooks failed")
		}
		return nil, err
	}
	specHooks.merge(addHooks)
	return specHooks, nil
}

// Create log file 
func createLogFile(logFilePath string) int {
	if logFilePath == "" {
		return 1
	}
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		err := os.MkdirAll(logFilePath, 0755)
		if err != nil {
			return 1
		}
	}
	logFileName := time.Now().Format("20060102") + ".log"
	logFileName = filepath.Join(logFilePath, logFileName)
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		return 1
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime)
	return 0
}
