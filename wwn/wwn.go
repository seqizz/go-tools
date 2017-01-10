package main

// gurkan.in | license: Apache License 2.0 shared on https://github.com/seqizz/go-tools/blob/master/LICENSE

import (
    "io/ioutil"
    "fmt"
    "strings"
    "os"
    "bufio"
    "github.com/olekukonko/tablewriter"
    "flag"
)

func main(){

    fcdir := "/sys/class/fc_host/"

    outputFlag := flag.String("output", "table", "Output style: table or plain")
    stateFlag := flag.String("state", "all", "State filter: online, offline or all")
    flag.Parse()

    files, err := ioutil.ReadDir(fcdir)
    if err != nil {
        fmt.Println("Couldn't find any FC interfaces")
        os.Exit(1)
    }

    table := tablewriter.NewWriter(os.Stdout)
    if *outputFlag == "table" {
        table.SetHeader([]string{"Name", "State", "Current Speed", "Supported Speeds", "WWN"})
    }

    for _, file := range files {
        state, _ := readLines(fcdir + file.Name()+ "/port_state")
        speed, _ := readLines(fcdir + file.Name()+ "/speed")
        suppspeed, _ := readLines(fcdir + file.Name()+ "/supported_speeds")
        wwn, _ := readLines(fcdir + file.Name()+ "/port_name")
        mid := []string{file.Name(), state[0], speed[0], suppspeed[0], wwn[0]}
        
        if *outputFlag == "table" {
                if *stateFlag == "all" {
                        table.Append(mid)
                } else if ( *stateFlag == "online" && state[0] == "Online" ) {
                        table.Append(mid)
                } else if ( *stateFlag == "offline" && state[0] == "Linkdown") {
                        table.Append(mid)
                }
        } else if *outputFlag == "plain" {
                pln := strings.Join(mid, ";")
                if *stateFlag == "all" {
                        fmt.Fprintln(os.Stdout, pln)
                } else if ( *stateFlag == "online" && state[0] == "Online" ) {
                        fmt.Fprintln(os.Stdout, pln)
                } else if ( *stateFlag == "offline" && state[0] == "Linkdown") {
                        fmt.Fprintln(os.Stdout, pln)
                }
        }
    }
    if *outputFlag == "table" {
            table.Render()
    }
    
}

func readLines(path string) ([]string, error) {
        file, err := os.Open(path)
        if err != nil {
                return nil, err
        }
        defer file.Close()

        var lines []string
        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
                lines = append(lines, scanner.Text())
        }
        return lines, scanner.Err()
}
