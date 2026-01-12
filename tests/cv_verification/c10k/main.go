package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

func main() {
	target := "localhost:8082"
	conns := 10000
	timeout := 10 * time.Second

	var wg sync.WaitGroup
	wg.Add(conns)

	success := 0
	var mu sync.Mutex

	fmt.Printf("Starting C10k test: Opening %d connections to %s...\n", conns, target)
	start := time.Now()

	for i := 0; i < conns; i++ {
		go func(id int) {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", target, timeout)
			if err != nil {
				return
			}
			defer conn.Close()

			// Send PING
			_, err = conn.Write([]byte("*1\r\n$4\r\nPING\r\n"))
			if err != nil {
				return
			}

			// Read PONG
			buf := make([]byte, 1024)
			conn.SetReadDeadline(time.Now().Add(timeout))
			_, err = conn.Read(buf)
			if err != nil {
				return
			}

			mu.Lock()
			success++
			mu.Unlock()
		}(i)

		// Throttle slightly to avoid running out of local ports/file descriptors instantly if ulimit is low
		if i%100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Printf("Finished C10k test in %v\n", duration)
	fmt.Printf("Successful connections/requests: %d/%d\n", success, conns)

	if success > 9000 {
		fmt.Println("PASS: C10k test passed (>90% success)")
	} else {
		fmt.Println("FAIL: C10k test failed")
	}
}
