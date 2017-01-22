package main

import (
	"fmt"
	"strconv"
	"time"
)

func minus(a, b int64) string {
	return strconv.FormatInt(a-b, 10)
}

func main() {

	var timestamp int64
	timestamp = 1485109537

	current := time.Now().Unix()

	// session := time.Unix(timestamp, 0).Format(time.RFC3339Nano)
	// current := time.Now().Format(time.RFC3339Nano)

	// Sun, 22 Jan 2017 12:28:20 +0000 (RFC)
	// 2017-01-22 12:27:57.005147257 +0000 GMT

	// diff := current.Sub(session).f
	// fmt.Println(diff)

	fmt.Println("session: ", timestamp)
	fmt.Println("current: ", current)

	fmt.Println(minus(timestamp, current))

}
