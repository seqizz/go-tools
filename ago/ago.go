package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

func relativeTime(timestamp int64) string {
	now := time.Now().Unix()
	diff := math.Abs(float64(timestamp - now))
	isFuture := timestamp > now

	var result string
	switch {
	case diff < 60:
		result = "Just now"
	case diff < 3600:
		result = fmt.Sprintf("%d minutes", int64(diff)/60)
	case diff < 86400:
		result = fmt.Sprintf("%d hours", int64(diff)/3600)
	case diff < 2592000:
		result = fmt.Sprintf("%d days", int64(diff)/86400)
	case diff < 31536000:
		result = fmt.Sprintf("%d months", int64(diff)/2592000)
	default:
		result = fmt.Sprintf("%d years", int64(diff)/31536000)
	}

	if isFuture && result != "Just now" {
		return "In " + result
	} else if !isFuture && result != "Just now" {
		return result + " ago"
	}
	return result
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide a Unix timestamp as an argument")
		os.Exit(1)
	}

	timestamp, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		fmt.Println("Invalid timestamp:", err)
		os.Exit(1)
	}

	fmt.Println(relativeTime(timestamp))
}
