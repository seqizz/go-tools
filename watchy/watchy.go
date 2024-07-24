package main

import (
    "bytes"
    "flag"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "github.com/nsf/termbox-go"
)

func main() {
    // Define and parse command-line flags
    interval := flag.Int("n", 2, "Specify update interval in seconds")
    flag.Parse()

    // Check if there's a command to run
    if len(flag.Args()) < 1 {
        fmt.Println("Usage: program -n <seconds> '<shell command>'")
        os.Exit(1)
    }

    // Get the user's shell, most meaningful way I can think of is $SHELL
    // and no '/bin/sh' or '/bin/bash' hardcoding is not good (see NixOS)
    shell := os.Getenv("SHELL")
    if shell == "" {
        fmt.Println("Error: Unable to determine shell. Please set the SHELL environment variable.")
        os.Exit(1)
    }

    err := termbox.Init()
    if err != nil {
        panic(err)
    }
    defer termbox.Close()

    eventQueue := make(chan termbox.Event)
    go func() {
        for {
            eventQueue <- termbox.PollEvent()
        }
    }()

    ticker := time.NewTicker(time.Duration(*interval) * time.Second)
    defer ticker.Stop()

    binaryName := filepath.Base(os.Args[0])
    command := strings.Join(flag.Args(), " ")
    hintLine := "Press Ctrl-C to quit, any other key to refresh instantly"

    update := func() {
        termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

        // Get current time for refresh timestamp
        refreshTime := time.Now().Format("15:04:05")

        // Display header with refresh time
        headerLine := fmt.Sprintf("%s: Running command every %d seconds: %s (last refresh: %s)",
                                  binaryName, *interval, command, refreshTime)
        displayLine(0, headerLine, termbox.ColorYellow, termbox.ColorDefault)
        displayLine(1, hintLine, termbox.ColorCyan, termbox.ColorDefault)

        // Execute the command and capture its output
        output, err := executeShellCommand(shell, command)
        if err != nil {
            output = []byte(err.Error())
        }

        // Display the output
        lines := splitLines(string(output))
        for y, line := range lines {
            displayLine(y+3, line, termbox.ColorDefault, termbox.ColorDefault)
        }

        termbox.Flush()
    }

    update()

    for {
        select {
        case ev := <-eventQueue:
            if ev.Type == termbox.EventKey {
                if ev.Key == termbox.KeyCtrlC {
                    return
                }
                update()
            }
        case <-ticker.C:
            update()
        }
    }
}

func executeShellCommand(shell, command string) ([]byte, error) {
    cmd := exec.Command(shell, "-c", command)
    var out bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = &out
    err := cmd.Run()
    return out.Bytes(), err
}

func splitLines(s string) []string {
    return strings.Split(strings.TrimSpace(s), "\n")
}

func displayLine(y int, s string, fg, bg termbox.Attribute) {
    for x, ch := range s {
        termbox.SetCell(x, y, ch, fg, bg)
    }
}
