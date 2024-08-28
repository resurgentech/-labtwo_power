package main

import (
	"fmt"
	"time"

	"github.com/bougou/go-ipmi"
)

func (h *Host) IpmiConnect() (*ipmi.Client, error) {
	ip := h.ipmi.ip
	port := h.ipmi.port
	username := h.ipmi.username
	password := h.ipmi.password

	client, err := ipmi.NewClient(ip, port, username, password)
	if err != nil {
		return nil, fmt.Errorf("error creating client for host %s: %v", h.hostname, err)
	}

	if *h.options.debug {
		client.WithDebug(true)
	}

	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("error connecting to host %s: %v", h.hostname, err)
	}
	return client, nil
}

func (h *Host) GetChassisPowerStatus(client *ipmi.Client) (bool, error) {
	cstatus, err := client.GetChassisStatus()
	if err != nil {
		return false, fmt.Errorf("error getting chassis status for host %s: %v", h.hostname, err)
	}
	chassisStatus := cstatus.PowerIsOn
	if *h.options.debug {
		fmt.Printf("Host %s chassis power status: %v\n", h.hostname, chassisStatus)
	}
	return chassisStatus, nil
}

func (h *Host) executeAction(action string) (bool, error) {
	client, err := h.IpmiConnect()
	if err != nil {
		return false, fmt.Errorf("error connecting to host %s: %v", h.hostname, err)
	}

	startingpowerstatus, err := h.GetChassisPowerStatus(client)
	if err != nil {
		return false, fmt.Errorf("error getting chassis status for host %s: %v", h.hostname, err)
	}

	desiredpowerstatus := false
	switch action {
	case "on":
		desiredpowerstatus = true
		if !startingpowerstatus {
			_, err = client.ChassisControl(ipmi.ChassisControlPowerUp)
		}
	case "off":
		if startingpowerstatus {
			_, err = client.ChassisControl(ipmi.ChassisControlPowerDown)
		}
	case "status":
		desiredpowerstatus = startingpowerstatus
		// Do nothing, just get the current power status
	default:
		return false, fmt.Errorf("invalid action: %s", action)
	}

	if err != nil {
		return false, fmt.Errorf("error executing action %s for host %s: %v", action, h.hostname, err)
	}

	// Wait for the power status to change if needed
	newpowerstatus := startingpowerstatus
	if startingpowerstatus != desiredpowerstatus {
		startTime := time.Now()
		for time.Since(startTime) < time.Duration(*h.options.timeout)*time.Second {
			time.Sleep(time.Duration(*h.options.ipmipolltime) * time.Second)
			powerstatus, err := h.GetChassisPowerStatus(client)
			if err != nil {
				return false, fmt.Errorf("error getting chassis status for host %s: %v", h.hostname, err)
			}
			if powerstatus == desiredpowerstatus {
				break
			}
		}
		newpowerstatus = desiredpowerstatus
	}

	return newpowerstatus, nil
}

/*
GetChassisStatus 	✓ 	chassis status, chassis power status
ChassisControl 	✓ 	chassis power on/off/cycle/reset/diag/soft
*/
