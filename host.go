package main

import "fmt"

type Ipmi struct {
	hostname string
	ip       string
	port     int
	username string
	password string
}

type Host struct {
	hostname string
	ip       string
	ipmi     Ipmi
	options  *Options
}

func (h *Host) New(options *Options, hostname string) error {
	h.options = options
	hostconfig, err := options.GetHostConfig(hostname)
	if err != nil {
		panic(fmt.Errorf("host %s not found in configuration should not be here", hostname))
	}
	// Adding the hostname or IP address to the hostname field
	h.hostname = hostname
	ip, ok := hostconfig["ip"].(string)
	if ok {
		h.ip = ip
	} else {
		// if we don't have the IP address, use the hostname
		h.ip = hostname
	}
	configpassword := false
	configusername := false
	ipmi, ok := hostconfig["ipmi"].(map[any]any)
	if ok {
		ipmihostname, ok := ipmi["hostname"].(string)
		if ok {
			h.ipmi.hostname = ipmihostname
		}
		ipmiip, ok := ipmi["ip"].(string)
		if ok {
			h.ipmi.ip = ipmiip
		} else {
			// if we don't have the IP address, use the hostname
			h.ipmi.ip = hostname
		}
		ipmiport, ok := ipmi["port"].(int)
		if ok {
			h.ipmi.port = ipmiport
		} else {
			h.ipmi.port = 623
		}
		ipmiusername, ok := ipmi["username"].(string)
		if ok {
			h.ipmi.username = ipmiusername
			configusername = true
		}
		ipmipassword, ok := ipmi["password"].(string)
		if ok {
			h.ipmi.password = ipmipassword
			configpassword = true
		}
	}
	if !(configusername && configpassword) {
		vaultusername, vaultpassword, err := GetHostCreds(options, hostname)
		if err != nil {
			return err
		}
		// config files, override vault creds
		if !configusername {
			h.ipmi.username = vaultusername
		}
		if !configpassword {
			h.ipmi.password = vaultpassword
		}
	}
	return nil
}
