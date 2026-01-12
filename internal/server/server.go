package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"github.com/Mazukiri/RedisClone/internal/core/io_multiplexing"
	"net/http"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/Mazukiri/RedisClone/internal/config"
	"github.com/Mazukiri/RedisClone/internal/constant"
	"github.com/Mazukiri/RedisClone/internal/core"
)

var eStatus int32 = constant.EngineStatusWaiting
var ClientCounter int32

const dashboardHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>MemKV Dashboard</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: sans-serif; padding: 20px; background: #f4f4f4; }
        .card { background: white; padding: 20px; margin-bottom: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; }
        .stat { font-size: 24px; font-weight: bold; color: #333; }
        .label { color: #666; font-size: 14px; }
        canvas { max-height: 300px; }
    </style>
</head>
<body>
    <h1>MemKV Dashboard</h1>
    <div class="grid">
        <div class="card">
            <div class="label">Connected Clients</div>
            <div id="clients" class="stat">0</div>
        </div>
        <div class="card">
            <div class="label">Memory Usage (MB)</div>
            <div id="memory" class="stat">0</div>
        </div>
        <div class="card">
            <div class="label">Total Keys</div>
            <div id="keys" class="stat">0</div>
        </div>
    </div>
    <div class="card">
        <canvas id="trafficChart"></canvas>
    </div>

    <script>
        const ctx = document.getElementById('trafficChart').getContext('2d');
        const chart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Connected Clients',
                    borderColor: 'rgb(75, 192, 192)',
                    data: [],
                    tension: 0.1
                }]
            },
            options: { animation: false }
        });

        async function update() {
            try {
                const res = await fetch('/metrics');
                const data = await res.json();
                
                document.getElementById('clients').innerText = data.clients;
                document.getElementById('memory').innerText = (data.memory_bytes / 1024 / 1024).toFixed(2);
                
                let totalKeys = 0;
                for (let k in data.keys) totalKeys += data.keys[k];
                document.getElementById('keys').innerText = totalKeys;

                const now = new Date().toLocaleTimeString();
                if (chart.data.labels.length > 20) {
                    chart.data.labels.shift();
                    chart.data.datasets[0].data.shift();
                }
                chart.data.labels.push(now);
                chart.data.datasets[0].data.push(data.clients);
                chart.update();
            } catch (e) { console.error(e); }
        }
        setInterval(update, 1000);
        update();
    </script>
</body>
</html>
`

func startMetricsServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(dashboardHTML))
	})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Handling /metrics request")
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		
		stats := map[string]interface{}{
			"clients":      atomic.LoadInt32(&ClientCounter),
			"memory_bytes": m.Alloc,
			"keys":         core.GetKeysCount(),
			"goroutines":   runtime.NumGoroutine(),
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	log.Printf("Starting Dashboard at http://0.0.0.0:%d\n", config.MetricsPort)
	go http.ListenAndServe(fmt.Sprintf(":%d", config.MetricsPort), nil)
}

func WaitForSignal(wg *sync.WaitGroup, signals chan os.Signal) {
	defer wg.Done()
	<-signals
	for atomic.LoadInt32(&eStatus) == constant.EngineStatusBusy {
	}
	log.Println("Shutting down gracefully")
	os.Exit(0)
}

func readCommandsFD(fd int) ([]*core.MemKVCmd, error) {
	var buf = make([]byte, 512)
	n, err := syscall.Read(fd, buf)
	if err != nil {
		return nil, err
	}
	
	var cmds []*core.MemKVCmd
	data := buf[:n]
	for len(data) > 0 {
		cmd, delta, err := core.ParseCmd(data)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
		data = data[delta:]
	}
	return cmds, nil
}

func responseRw(cmd *core.MemKVCmd, rw io.ReadWriter) {
	err := core.EvalAndResponse(cmd, rw)
	if err != nil {
		responseErrorRw(err, rw)
	}
}

func responseErrorRw(err error, rw io.ReadWriter) {
	rw.Write([]byte(fmt.Sprintf("-%s%s", err, core.CRLF)))
}

func RunAsyncTCPServer(wg *sync.WaitGroup) error {
	defer wg.Done()
	log.Println("starting an asynchronous TCP server on", config.Host, config.Port)

	// Start HTTP Metrics Dashboard
	startMetricsServer()

	var events = make([]io_multiplexing.Event, config.MaxConnection)

	// Create a server socket. A socket is an endpoint for communication between client and server
	serverFD, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Println(err)
		return err
	}
	defer syscall.Close(serverFD)

	// Set the Socket operate in a non-blocking mode
	// Default mode is blocking mode: when you read from a FD, control isn't returned
	// until at least one byte of data is read.
	// Non-blocking mode: if the read buffer is empty, it will return immediately.
	// We want non-blocking mode because we will use epoll to monitor and then read from
	// multiple FD, so we want to ensure that none of them cause the program to "lock up."
	if err = syscall.SetNonblock(serverFD, true); err != nil {
		log.Println(err)
		return err
	}

	if err = syscall.SetsockoptInt(serverFD, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		log.Println(err)
		return err
	}

	// Bind the IP and the port to the server socket FD.
	ip4 := net.ParseIP(config.Host)
	if err = syscall.Bind(serverFD, &syscall.SockaddrInet4{
		Port: config.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	}); err != nil {
		log.Println(err)
		return err
	}

	// Start listening
	if err = syscall.Listen(serverFD, config.MaxConnection); err != nil {
		log.Println(err)
		return err
	}

	// ioMultiplexer is an object that can monitor multiple file descriptor (FD) at the same time.
	// When one or more monitored FD(s) are ready for IO, it will notify our server.
	// Here, we use ioMultiplexer to monitor Server FD and Clients FD.
	ioMultiplexer, err := io_multiplexing.CreateIOMultiplexer()
	if err != nil {
		return err
	}
	defer ioMultiplexer.Close()

	// Monitor "read" events on the Server FD
	if err = ioMultiplexer.Monitor(io_multiplexing.Event{
		Fd: serverFD,
		Op: io_multiplexing.OpRead,
	}); err != nil {
		return err
	}

	for atomic.LoadInt32(&eStatus) != constant.EngineStatusShuttingDown {
		// check if any FD is ready for an IO
		events, err = ioMultiplexer.Check()
		if err != nil {
			continue
		}

		if !atomic.CompareAndSwapInt32(&eStatus, constant.EngineStatusWaiting, constant.EngineStatusBusy) {
			if eStatus == constant.EngineStatusShuttingDown {
				return nil
			}
		}
		for i := 0; i < len(events); i++ {
			if events[i].Fd == serverFD {
				// the Server FD is ready for reading, means we have a new client.
				atomic.AddInt32(&ClientCounter, 1)
				log.Printf("new client: id=%d\n", atomic.LoadInt32(&ClientCounter))
				// accept the incoming connection from a client
				connFD, _, err := syscall.Accept(serverFD)
				if err != nil {
					log.Println("err", err)
					continue
				}

				if err = syscall.SetNonblock(connFD, true); err != nil {
					return err
				}

				// add this new connection to be monitored
				if err = ioMultiplexer.Monitor(io_multiplexing.Event{
					Fd: connFD,
					Op: io_multiplexing.OpRead,
				}); err != nil {
					return err
				}
			} else {
				// the Client FD is ready for reading, means an existing client is sending a command
				comm := core.FDComm{Fd: int(events[i].Fd)}
				cmds, err := readCommandsFD(comm.Fd)
				if err != nil {
					syscall.Close(events[i].Fd)
					atomic.AddInt32(&ClientCounter, -1)
					log.Println("client quit")
					atomic.SwapInt32(&eStatus, constant.EngineStatusWaiting)
					continue
				}
				for _, cmd := range cmds {
					responseRw(cmd, comm)
				}
			}
			atomic.SwapInt32(&eStatus, constant.EngineStatusWaiting)
		}
	}

	return nil
}
