package main

// gurkan.in | license: Apache License 2.0 shared on https://github.com/seqizz/go-tools/blob/master/LICENSE

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/cheggaaa/pb"
	"github.com/spf13/viper"
)

var work string
var runslice []string
var f *os.File
var err error
var parallel int
var noWrite = false
var isVerbose = false
var isScript = false

//Variables , why not struct
type Variables struct {
	server string
	work   string
	config *viper.Viper
	bar    *pb.ProgressBar
	file   *os.File
	auth   []ssh.AuthMethod
}

func main() {
	configFile := flag.String("config", "/home/gurkan/go/sshit/config.yaml", "Configuration file to use")
	serverName := flag.String("server", "", "Server or server group to run commands on")
	commandToRun := flag.String("command", "", "Command to run on remote server(s)")
	scriptToRun := flag.String("script", "", "Script to run on remote server(s)")
	outputFlag := flag.String("output", "", "Result file to write all responses/errors")
	timeoutFlag := flag.Int("timeout", 0, "Maximum allowed time for running the command")
	showNewConfig := flag.Bool("C", false, "Show a config file example")
	flag.Parse()

	conf := viper.New()
	if !CheckIfFileExists(*configFile) {
		sendError("config file")
	}
	conf.SetConfigFile(*configFile)
	err := conf.ReadInConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Can't read config file: "+*configFile+"\n", err)
		fmt.Println("You can create a new config file template with -C flag.")
		os.Exit(1)
	}

	checkConfigFlag(*showNewConfig)

	wg := sync.WaitGroup{}

	if conf.GetString("parallel") != "" {
		if _, err := strconv.Atoi(conf.GetString("parallel")); err == nil {
			parallel, _ = strconv.Atoi(conf.GetString("parallel"))
		} else {
			fmt.Fprintln(os.Stderr, "Didn't understand 'parallel' line from config file, give me numbers!")
			os.Exit(1)
		}
	} else {
		parallel = 1
	}

	wg.Add(parallel)
	s := make(chan string, parallel)

	if conf.GetBool("verbose") {
		isVerbose = true
	}

	//Should I write it to a file?
	if *outputFlag == "" {
		if conf.GetString("outputfile") == "" {
			noWrite = true
		} else {
			*outputFlag = conf.GetString("outputfile")
		}
	}

	if !noWrite {
		f, err = os.OpenFile(*outputFlag, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR: Can't write to the output file!\n", err)
			os.Exit(1)
		}
		defer defClose(f)
	}

	if *serverName == "" {
		sendError("server/group name")
	}
	//Checking if it is a group
	if strings.Contains(*serverName, "group:") {
		tempgrp := strings.Replace(*serverName, "group:", "groups.", -1)
		if len(conf.GetStringSlice(tempgrp)) == 0 {
			sendError("valid group name")
		}
		runslice = conf.GetStringSlice(tempgrp)
	} else {
		runslice = append(runslice, *serverName)
	}

	// Is there a command?
	if *commandToRun == "" && *scriptToRun == "" {
		sendError("command or script")
	} else {
		if *commandToRun != "" && *scriptToRun != "" {
			sendError("single thing; command and script can't be used at the same time.")
		}
		if *commandToRun != "" {
			work = *commandToRun
		} else {
			work = *scriptToRun
			isScript = true
		}
	}

	//Configure the status bar
	bar := pb.New(len(runslice))
	if !isVerbose {
		bar.ShowTimeLeft = false
		bar.ShowSpeed = false
		bar.Start()
	}

	//Shit is getting real
	for i := 0; i < parallel; i++ {
		go func() {
			defer wg.Done()
			for server := range s {
				mySet := &Variables{
					server: server,
					work:   work,
					config: conf,
					bar:    nil,
					file:   f,
					auth:   []ssh.AuthMethod{PublicKeyFile(conf.GetString("sshkey"))},
				}
				if isVerbose {
					response, err := checkTimeout(mySet, time.Duration(int64(*timeoutFlag)))
					if err != nil {
						fmt.Println("ERROR -", server, ":", err)
					} else {
						fmt.Fprint(os.Stdout, server, ":", response)
					}
				} else {
					mySet.bar = bar
					checkTimeout(mySet, time.Duration(int64(*timeoutFlag)))
				}
			}
		}()
	}

	//Feed the channel
	for _, server := range runslice {
		s <- server
	}

	close(s)

	wg.Wait()
}

//runCommand ... as name suggests , might be something about running a command
func runCommand(set *Variables) (response string, err error) {

	if set.bar != nil {
		set.bar.Increment()
	}

	sshConfig := &ssh.ClientConfig{
		User: set.config.GetString("username"),
		Auth: set.auth,
	}

	connection, err := ssh.Dial("tcp", set.server+":22", sshConfig)
	if err != nil {
		return "Failed to dial", err
	}

	session, err := connection.NewSession()
	if err != nil {
		return "Failed to create session", err
	}

	var answer []byte

	answer, err = session.Output(set.work)
	if err != nil {
		if !noWrite {
			_, writeErr := set.file.WriteString("ERROR : " + set.server + " : " + err.Error() + "\n")
			if writeErr != nil {
				fmt.Fprintln(os.Stderr, "Couldn't write to output file")
			}
		}
		return "", err
	}

	defer session.Close()

	if !noWrite {
		_, writeErr := set.file.WriteString(set.server + " : " + string(answer))
		if writeErr != nil {
			fmt.Fprintln(os.Stderr, "Couldn't write to output file", writeErr)
		}
	}

	return string(answer), nil
}

//Just close the file please
func defClose(file *os.File) {
	err := file.Close()
	if err != nil {
		log.Fatal(err)
	}
}

//I keep saying the same thing, so why not a func?
func sendError(z string) {
	fmt.Println("Please give a", z)
	os.Exit(1)
}

func checkTimeout(set *Variables, timeout time.Duration) (response string, err error) {

	if timeout == 0 {
		timeout = 999999999999
	}

	var resp string

	timechannel := make(chan bool, 1)
	answerchannel := make(chan bool, 1)
	go func() {
		time.Sleep(time.Second * timeout)
		timechannel <- true
	}()

	go func() {
		resp, err = runCommand(set)
		answerchannel <- true
	}()

	select {
	case t := <-timechannel:
		timeError := errors.New("Taking more than timeout, aborted")
		_ = t
		return "", timeError
	case o := <-answerchannel:
		_ = o
		return resp, err
	}
}

//CheckIfFileExists checks the file is present on the disk
func CheckIfFileExists(filePath string) bool {
	if _, err := os.Stat(filePath); err == nil {
		return true
	}
	return false
}

//PublicKeyFile is reading file and shitting as AuthMethod
func PublicKeyFile(file string) ssh.AuthMethod {

	if !CheckIfFileExists(file) {
		sendError("valid public key path")
	}

	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func checkConfigFlag(display bool) {
	if display {
		fmt.Println(`
#Config file example
sshkey: /home/user/.ssh/id_rsa
username: root
port: 22
outputfile: results.txt
verbose: true
parallel: 5

groups:
  webservers:
   - 127.0.0.1
   - 127.0.0.2
   - 127.0.0.3
   - 127.0.0.4
  dbservers:
   - 127.0.1.1
   - 127.0.2.2
   - 127.0.3.3`)
		os.Exit(0)
	}
}

