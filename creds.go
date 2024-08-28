package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	// 	"log"
	// 	"time"

	"github.com/hashicorp/vault-client-go"
	// "github.com/hashicorp/vault-client-go/schema"
)

func GetVaultCreds(vaultconfig map[any]any, keypath string) (map[string]any, error) {
	ctx := context.Background()
	token := vaultconfig["token"].(string)
	hostname := vaultconfig["hostname"].(string)
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
		return nil, err
	}

	if err := client.SetToken(token); err != nil {
		return nil, err
	}

	// read the secret
	s, err := client.Secrets.KvV1Read(ctx, secret, vault.WithMountPath(path))
	if err != nil {
		return nil, err
	}

	return s.Data, nil
}

func GetHostCreds(options *Options, hostname string) (string, string, error) {
	vaultconfig, err := options.GetVaultConfig()
	if err != nil {
		return "", "", err
	}
	keypathraw, ok := vaultconfig["path"]
	if !ok {
		return "", "", fmt.Errorf("path not found in vault configuration")
	}
	keypath, ok := keypathraw.(string)
	if !ok {
		return "", "", fmt.Errorf("path not a string in vault configuration")
	}

	creds, err := GetVaultCreds(vaultconfig, keypath)
	if err != nil {
		return "", "", err
	}
	hostlistraw, ok := creds["host"]
	if !ok {
		return "", "", fmt.Errorf("host not found in secret at %s", keypath)
	}
	hostlist, ok := hostlistraw.([]map[string]any)
	if !ok {
		return "", "", fmt.Errorf("hosts list malformed, expecting array %s", keypath)
	}
	for _, host := range hostlist {
		hostname, ok := host["hostname"].(string)
		if !ok {
			return "", "", fmt.Errorf("hostname not found in secret at %s", keypath)
		}
		if hostname == hostname {
			username, ok := host["username"].(string)
			if !ok {
				return "", "", fmt.Errorf("username not found in secret at %s", keypath)
			}
			password, ok := host["password"].(string)
			if !ok {
				return "", "", fmt.Errorf("password not found in secret at %s", keypath)
			}
			return username, password, nil
		}
	}
	return "", "", fmt.Errorf("host creds for %s not found in secret at %s", hostname, keypath)
}
