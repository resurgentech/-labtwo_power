package main

import (
	"fmt"
	"os"
	"strings"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

/*
Options holds the configuration of this run.
*/
type Options struct {
	hosts          []string    // List of hosts to process
	action         *string     // From CLI - action to execute
	debug          *bool       // Enable debug mode
	noninteractive *bool       // Disable interactive mode
	ipmipolltime   *int        // IPMI poll time in seconds
	timeout        *int        // Timeout for the action in seconds
	retries        *int        // Number of retries
	vaulttoken     *string     // Vault token
	configfile     *string     // Path to the configuration file
	config         map[any]any // structure from yaml configfile
}

func (o *Options) ParseCli() error {
	local_hosts := flag.StringP("hosts", "h", "", "List of hosts to act on (comma separated)")
	o.action = flag.StringP("action", "a", "status", "up, down, or status (default: status)")
	o.debug = flag.BoolP("debug", "d", false, "Prints extra debug information")
	o.noninteractive = flag.BoolP("noninteractive", "n", false, "Disable interactive mode")
	o.ipmipolltime = flag.IntP("ipmipolltime", "p", 5, "IPMI poll time in seconds")
	o.timeout = flag.IntP("timeout", "o", 60, "Timeout for the action in seconds")
	o.retries = flag.IntP("retries", "r", 3, "Number of retries for api calls")
	o.vaulttoken = flag.StringP("token", "t", "", "Vault/OpenBao token")
	o.configfile = flag.StringP("config", "c", "./config.yml,~/.labtwo/config.yaml,/etc/labtwo/config.yaml", "Path to the configuration file overides the defaults")
	flag.Parse()

	// Add hosts from CLI
	err := o.ParseHosts(local_hosts)
	if err != nil {
		return err
	}

	// Parse the configuration file
	err = o.ParseConfigFile()
	if err != nil {
		return err
	}

	return nil
}

func (o *Options) ParseHosts(local_hosts *string) error {
	if *local_hosts == "" {
		return fmt.Errorf("no hosts specified. use -h or --hosts flag")
	}
	for _, host := range strings.Split(*local_hosts, ",") {
		if *o.debug {
			fmt.Printf("Adding host '%s'\n", host)
		}
		o.hosts = append(o.hosts, strings.TrimSpace(host))
	}
	return nil
}

func (o *Options) ParseConfigFile() error {
	o.config = make(map[any]any)

	// Iterate through the list of configuration files
	for _, configFile := range strings.Split(*o.configfile, ",") {
		// Check if the file exists
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			if *o.debug {
				fmt.Printf("Config file %s does not exist\n", configFile)
			}
			continue
		}
		// Read and parse the YAML file
		data, err := os.ReadFile(configFile)
		if err != nil {
			if *o.debug {
				fmt.Printf("Error reading config file %s: %v\n", configFile, err)
			}
			continue
		}

		if *o.debug {
			fmt.Printf("Parsing config file %s\n", configFile)
		}

		err = yaml.Unmarshal(data, &o.config)
		if err != nil {
			if *o.debug {
				fmt.Printf("Error parsing YAML in config file %s: %v\n", configFile, err)
			}
			continue
		}

		// Successfully parsed a config file, no need to continue
		o.configfile = &configFile
		break
	}
	if len(o.config) == 0 {
		return fmt.Errorf("no valid configuration file found")
	}

	// Validate the configuration
	err := o.ValidateConfig()
	if err != nil {
		return err
	}

	return nil
}

func (o *Options) ValidateConfig() error {
	// Validate the configuration
	err := o.ValidateConfigHosts()
	if err != nil {
		return err
	}
	err = o.ValidateConfigVault()
	if err != nil {
		return err
	}
	return nil
}

func (o *Options) GetHostConfig(hostname string) (map[any]any, error) {
	// Confirm that each host is listed in the configuration file
	hostsraw, ok := o.config["hosts"]
	if !ok {
		return nil, fmt.Errorf("'hosts' key not found in configuration")
	}
	hosts, ok := hostsraw.([]any)
	if !ok {
		return nil, fmt.Errorf("'hosts' is not a map in configuration")
	}
	for _, hostraw := range hosts {
		host, ok := hostraw.(map[any]any)
		if !ok {
			return nil, fmt.Errorf("skipping invalid host config entry: %v", hostraw)
		}
		if host["hostname"] == hostname {
			return host, nil
		}
	}
	return nil, fmt.Errorf("host %s not found in configuration", hostname)
}

func (o *Options) GetVaultConfig() (map[any]any, error) {
	// Confirm that each host is listed in the configuration file
	vaultraw, ok := o.config["vault"]
	if !ok {
		return nil, fmt.Errorf("'vault' key not found in configuration")
	}
	vault, ok := vaultraw.(map[any]any)
	if !ok {
		return nil, fmt.Errorf("'vault' is not a map in configuration")
	}
	return vault, nil
}

func (o *Options) ValidateConfigHosts() error {
	// Confirm that each host is listed in the configuration file
	hosts, ok := o.config["hosts"]
	if !ok {
		return fmt.Errorf("'hosts' key not found in configuration")
	}

	hostsList, ok := hosts.([]any)
	if !ok {
		return fmt.Errorf("'hosts' is not a list in configuration")
	}

	var hostsArray []string

	for _, host := range o.hosts {
		for _, hostConfigInterface := range hostsList {
			hostConfig, ok := hostConfigInterface.(map[any]any)
			if !ok {
				if *o.debug {
					fmt.Printf("Skipping invalid host config entry: %v\n", hostConfigInterface)
				}
				continue
			}

			hostnameInterface, ok := hostConfig["hostname"]
			if !ok {
				if *o.debug {
					fmt.Printf("Host config missing 'hostname': %v\n", hostConfig)
				}
				continue
			}

			hostname, ok := hostnameInterface.(string)
			if !ok {
				if *o.debug {
					fmt.Printf("'hostname' is not a string: %v\n", hostnameInterface)
				}
				continue
			}

			if *o.debug {
				fmt.Printf("Checking host '%s' against config host '%s'\n", host, hostname)
			}

			if host == hostname {
				hostsArray = append(hostsArray, hostname)
				break
			}

			hostnameStripped := strings.Split(hostname, ".")[0]
			if *o.debug {
				fmt.Printf("Checking host '%s' against stripped config host '%s'\n", host, hostnameStripped)
			}
			if host == hostnameStripped {
				hostsArray = append(hostsArray, hostname)
				break
			}
		}
	}
	if len(hostsArray) == 0 {
		return fmt.Errorf("no hosts found in configuration file")
	}
	o.hosts = hostsArray

	return nil
}

func (o *Options) ValidateConfigVault() error {

	vaultconfig, err := o.GetVaultConfig()
	if err != nil {
		return err
	}

	_, ok := vaultconfig["hostname"]
	if !ok {
		return fmt.Errorf("'vault.hostname' key not found in configuration %s", *o.configfile)
	}
	_, ok = vaultconfig["path"]
	if !ok {
		return fmt.Errorf("'vault.path' key not found in configuration %s", *o.configfile)
	}
	_, ok = vaultconfig["token"]
	if ok {
		if *o.vaulttoken != "" {
			if *o.debug {
				fmt.Printf("Vault token is set in configuration file and the cli\nUsing the cli token\n")
			}
			vaultconfig["token"] = *o.vaulttoken
		}
	} else {
		if *o.debug {
			fmt.Printf("Vault token is not set in configuration file nor the cli\nChecking envvar VAULTTOKEN\n")
		}
		if os.Getenv("VAULTTOKEN") != "" {
			if *o.debug {
				fmt.Printf("Vault token is set in envvar VAULTTOKEN\n")
			}
			vaultconfig["token"] = os.Getenv("VAULTTOKEN")
		} else {
			if !*o.noninteractive {
				if *o.debug {
					fmt.Printf("Vault token is not set in envvar VAULTTOKEN\nAsking user")
				}
				var vaulttoken string
				fmt.Printf("Enter vault token: ")
				fmt.Scanln(&vaulttoken)
				vaultconfig["token"] = vaulttoken
			} else {
				return fmt.Errorf("can't find vault token")
			}
		}
	}

	return nil
}
