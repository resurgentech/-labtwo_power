package main

func main() {
	var options Options

	err := options.ParseCli()
	if err != nil {
		panic(err)
	}

	for _, hostname := range options.hosts {
		var host Host
		host.New(&options, hostname)
		host.executeAction(*options.action)
	}
}
