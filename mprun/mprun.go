package main

// gurkan.in | license: Apache License 2.0 shared on https://github.com/seqizz/go-tools/blob/master/LICENSE

import (
	"fmt"
	"strings"
	"sync"
	//        "time"
	"bufio"
	"bytes"
	"flag"
	"os"
	"os/exec"

	"github.com/cheggaaa/pb"
)

func main() {

	lineFlag := flag.String("inputfile", "inputfile.input", "File to read inputs")
	commandFlag := flag.String("script", "my.script", "File to run")
	runnerFlag := flag.String("runner", "bash", "Script interpreter")
	outputFlag := flag.String("output", "results.txt", "Result file")
	workerFlag := flag.Int("workers", 2, "Worker thread count")
	verboseFlag := flag.Bool("verbose", false, "Print output to stdout")
	flag.Parse()

	if (len(os.Args) < 2) || (strings.Contains(os.Args[1], "-help")) {
		howto()
	}

	if _, err := os.Stat(*lineFlag); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Couldn't find inputfile:", *lineFlag)
		os.Exit(1)
	}
	if _, err := os.Stat(*commandFlag); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Couldn't find script:", *commandFlag)
		os.Exit(1)
	}

	lines, err := readLines(*lineFlag)
	errcontrol(err)
	bar := pb.New(len(lines))
	bar.ShowTimeLeft = false
	bar.ShowSpeed = false

	f, err := os.OpenFile(*outputFlag, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	errcontrol(err)
	defer defClose(f)

	// _ = "breakpoint"

	c := make(chan string, *workerFlag)

	wg := sync.WaitGroup{}
	wg.Add(*workerFlag)
	for i := 0; i < *workerFlag; i++ {
		go func() {
			defer wg.Done()
			if !*verboseFlag {
				Process(c, *commandFlag, f, *runnerFlag, *verboseFlag, bar)
			} else {
				Process(c, *commandFlag, f, *runnerFlag, *verboseFlag, nil)
			}
		}()
	}

	for _, hostname := range lines {
		c <- hostname
	}

	close(c)

	wg.Wait()
}

// Process : Actual working process for one thread
func Process(c chan string, command string, file *os.File, interpreter string, isVerbose bool, progBar *pb.ProgressBar) {
	for hostname := range c {
		var out bytes.Buffer
		//      time.Sleep(time.Millisecond * 1000)
		if !isVerbose {
			progBar.Start()
			progBar.Increment()
		} else {
			fmt.Printf("Processing: %s\n", hostname)
		}
		cmd := exec.Command(interpreter, command, hostname)
		cmd.Stdout = &out
		err := cmd.Run()
		errcontrol(err)
		_, writeErr := file.WriteString(out.String() + "\n")
		errcontrol(writeErr)

		if isVerbose {
			fmt.Fprintln(os.Stdout, out.String())
		}
	}
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer defClose(file)

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()

}

func howto() {
	fmt.Println(`
        Welcome to the Miserable Parallel Runner
        
        This program runs given script in multiple threads, and feeds every one of them a line from a text file.

        Example:
        ./mprun -workers=3 -inputfile=myhostfile -script=myscript.py -runner=python3 -output=results.csv -verbose

        This will run myscript.py n times (n = line count of "myhostfile")
        

        workers:        Worker thread count to run parallel
        inputfile:      File (hostnames/IP addresses etc.) to feed into script
        script:         Main file to run
        runner:         Interpreter for 'script' (python2, python3, bash etc.) 
        output:         File to write results in
        verbose:        Prints script's output to both file and stdout
        `)
	os.Exit(0)
}

func defClose(file *os.File) {
	err := file.Close()
	if err != nil {
		fmt.Fprintln(os.Stdout, "Failed to close file")
	}
}

func errcontrol(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
	}
}
