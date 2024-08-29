package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/vault-client-go"
)

func ReadVault(client *vault.Client, secret string, path string) (map[string]any, error) {
	ctx := context.Background()
	s, err := client.Secrets.KvV1Read(ctx, secret, vault.WithMountPath(path))
	if err != nil {
		return nil, err
	}
	return s.Data, nil
}

func GetVaultCreds(options *Options) (map[string]any, string, error) {
	vaultconfig, err := options.GetVaultConfig()
	if err != nil {
		return nil, "", err
	}
	tokenraw, ok := vaultconfig["token"]
	if !ok {
		return nil, "", fmt.Errorf("token not found in vault configuration")
	}
	token, ok := tokenraw.(string)
	if !ok {
		return nil, "", fmt.Errorf("token not a string in vault configuration")
	}
	hostnameraw, ok := vaultconfig["hostname"]
	if !ok {
		return nil, "", fmt.Errorf("hostname not found in vault configuration")
	}
	hostname, ok := hostnameraw.(string)
	if !ok {
		return nil, "", fmt.Errorf("hostname not a string in vault configuration")
	}
	keypathraw, ok := vaultconfig["path"]
	if !ok {
		return nil, "", fmt.Errorf("path not found in vault configuration")
	}
	keypath, ok := keypathraw.(string)
	if !ok {
		return nil, "", fmt.Errorf("path not a string in vault configuration")
	}

	parts := strings.Split(keypath, "/")
	secret := parts[len(parts)-1]
	path := strings.Join(parts[:len(parts)-1], "/")
	tlsconfig := vault.TLSConfiguration{
		InsecureSkipVerify: true,
	}
	client, err := vault.New(
		vault.WithAddress(hostname),
		vault.WithRequestTimeout(10*time.Second),
		vault.WithTLS(tlsconfig),
	)
	if err != nil {
		return nil, "", err
	}

	if err := client.SetToken(token); err != nil {
		return nil, "", err
	}

	// read the secret
	retries := *options.retries
	var creds map[string]any
	for retries > 0 {
		creds, err = ReadVault(client, secret, path)
		if err == nil {
			break
		}
		retries--
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		return nil, "", err
	}

	return creds, keypath, nil
}

func GetHostCreds(options *Options, hostname string) (string, string, error) {
	creds, keypath, err := GetVaultCreds(options)
	if err != nil {
		return "", "", err
	}
	hostlistraw, ok := creds["hosts"]
	if !ok {
		return "", "", fmt.Errorf("top level 'hosts' key not found in secret at %s", keypath)
	}
	hostlist, ok := hostlistraw.([]any)
	if !ok {
		fmt.Printf("the class of hostlistraw is: %T\n", hostlistraw)
		return "", "", fmt.Errorf("'hosts' list malformed, expecting array %s", keypath)
	}
	for _, hostraw := range hostlist {
		host, ok := hostraw.(map[string]any)
		if !ok {
			return "", "", fmt.Errorf("host malformed at %s", keypath)
		}
		chostnameraw, ok := host["hostname"]
		if !ok {
			return "", "", fmt.Errorf("'hostname' key not found in secret at %s", keypath)
		}
		chostname, ok := chostnameraw.(string)
		if !ok {
			return "", "", fmt.Errorf("'hostname' not a string in secret at %s", keypath)
		}
		if chostname != hostname {
			continue
		}
		ipmiraw, ok := host["ipmi"]
		if !ok {
			fmt.Printf("host %v\n", host)
			return "", "", fmt.Errorf("'ipmi' key not found in secret at %s", keypath)
		}
		ipmi, ok := ipmiraw.(map[string]any)
		if !ok {
			return "", "", fmt.Errorf("'ipmi' malformed at %s", keypath)
		}
		usernameraw, ok := ipmi["username"]
		if !ok {
			return "", "", fmt.Errorf("'username' key not found in 'ipmi' secret at %s", keypath)
		}
		username, ok := usernameraw.(string)
		if !ok {
			return "", "", fmt.Errorf("'username' not a string in 'ipmi' secret at %s", keypath)
		}
		passwordraw, ok := ipmi["password"]
		if !ok {
			return "", "", fmt.Errorf("'password' key not found in 'ipmi' secret at %s", keypath)
		}
		password, ok := passwordraw.(string)
		if !ok {
			return "", "", fmt.Errorf("'password' not a string in 'ipmi' secret at %s", keypath)
		}
		return username, password, nil
	}
	return "", "", fmt.Errorf("host creds for %s not found in secret at %s", hostname, keypath)
}
