package main

import (
	"fmt"
	"sync"
)

type actionoutput struct {
	status bool
	err    error
}

func main() {
	var options Options

	err := options.ParseCli()
	if err != nil {
		panic(err)
	}

	output := make(map[string]actionoutput)
	hosts := make(map[string]Host)
	for _, hostname := range options.hosts {
		var host Host
		err := host.New(&options, hostname)
		if err != nil {
			fmt.Printf("Error main - Host %s\n", hostname)
			panic(err)
		}
		if *options.debug {
			fmt.Printf("Setting up host map: %s\n", hostname)
		}
		hosts[hostname] = host
	}
	var wg sync.WaitGroup
	for hostname, host := range hosts {
		wg.Add(1)
		go func(hostname string, host Host) {
			defer wg.Done()
			status, err := host.executeAction(*options.action)
			output[hostname] = actionoutput{status, err}
		}(hostname, host)
	}
	wg.Wait()
	for hostname, o := range output {
		if o.err != nil {
			fmt.Printf("Error main - Host %s: '%v'\n", hostname, o.err)
		} else {
			fmt.Printf("Host '%s' power: %s\n", hostname, map[bool]string{true: "ON", false: "OFF"}[o.status])
		}
	}
}
