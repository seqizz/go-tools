package main

// gurkan.in | license: Apache License 2.0 shared on https://github.com/seqizz/go-tools/blob/master/LICENSE

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/labstack/gommon/bytes"
)

var baseLevel, curLevel, count int
var treshold, biggest, tmpGuiltySize int64
var champ, tmpGuilty string
var guilty []string
var guiltySize []int64

func main() {
	cmdName := "du"

	var cmdArgs []string
	allInOne := make(map[string]int64)
	var wg sync.WaitGroup

	if len(os.Args) == 1 || len(os.Args) > 3 {
		usage()
	}

	if _, err := os.Stat(os.Args[1]); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Couldn't find:", os.Args[1])
		os.Exit(1)
	}

	rootFolder := os.Args[1]

	baseLevel = strings.Count(rootFolder, "/")

	if runtime.GOOS == "linux" {
		cmdArgs = []string{"-x", rootFolder}
	} else if runtime.GOOS == "solaris" {
		cmdArgs = []string{"-d", rootFolder}
	}

	cmd := exec.Command(cmdName, cmdArgs...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for Cmd", err)
		os.Exit(1)
	}

	wg.Add(1)

	scanner := bufio.NewScanner(cmdReader)

	go runCommand(scanner, allInOne, &wg)

	err = cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
		os.Exit(1)
	}

	wg.Wait()

	isDone := false

	curLevel = baseLevel + 1

	treshold = 30

	if len(os.Args) == 3 {
		count, err = strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		count = 3
	}

	for i := 0; i < count; i++ {
		guiltySize[i], guilty[i] = calculate(allInOne, curLevel, isDone, i)
		s := bytes.Format(guiltySize[i] * 1024)

		fmt.Println(guilty[i], s)
	}

}

func usage() {
	fmt.Fprint(os.Stdout, "\nKahin finds the most revelant space-hugger directories under given path\n(and does not calculate other mountpoints found).\n\nUsage:\n\tkahin {directory} {count}\n\nFor example, if you want to find 5 most revelant directories, use:\n\n\tkahin /data 5\n\n")
	os.Exit(1)
}

func runCommand(scanner *bufio.Scanner, allInOne map[string]int64, wg *sync.WaitGroup) {
	for scanner.Scan() {
		splitted := strings.Split(scanner.Text(), "\t")
		myint, err := strconv.ParseInt(splitted[0], 10, 64)
		if err == nil {
			allInOne[splitted[1]] = myint
		}
		//		fmt.Printf("command output | %s\n", scanner.Text())
		//		fmt.Printf("checking if map is ok | %d\n", allInOne[splitted[1]])
	}
	wg.Done()
}

func levelCount(str string) int {
	level := strings.Count(str, "/")
	return level
}

func calculate(myMap map[string]int64, curLevel int, isDone bool, number int) (int64, string) {
	if isDone {
		guilty = append(guilty, tmpGuilty)
		guiltySize = append(guiltySize, tmpGuiltySize)
		tmpGuilty = ""
		tmpGuiltySize = 0
		treshold = 30
		curLevel = baseLevel + 1
		return guiltySize[number], guilty[number]
	}

	biggest = 0
	champ = ""
	tempMap := make(map[string]int64)

	for k, v := range myMap {
		if levelCount(k) == curLevel {
			if !isSliceContainsRgx(guilty, k) {
				tempMap[k] = v
			}
		}
	}

	for k, v := range tempMap {
		if v > biggest {
			biggest = v
			champ = k
		}
	}
	if biggest > tmpGuiltySize/100*treshold {
		tmpGuiltySize = biggest
		tmpGuilty = champ
		curLevel++
		treshold = treshold + 10
	} else {
		isDone = true
	}

	return calculate(myMap, curLevel, isDone, number)
}

func isSliceContainsRgx(slice []string, text string) bool {
	for _, a := range slice {
		a = strings.Replace(a, "(", "\\(", -1)
		a = strings.Replace(a, ")", "\\)", -1)
		a = strings.Replace(a, "[", "\\[", -1)
		a = strings.Replace(a, "]", "\\]", -1)
		matched, _ := regexp.MatchString(a+".*", text)
		matchedReverse, _ := regexp.MatchString(text, a+".*")
		if matched || matchedReverse {
			return true
		}
	}
	return false
}
