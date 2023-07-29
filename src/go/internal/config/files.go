package config

import (
	"os"
	"path/filepath"
)

var (
	CAFile               = configFile("CERT_DIR", "ca.pem")
	ServerCertFile       = configFile("CERT_DIR", "server.pem")
	ServerKeyFile        = configFile("CERT_DIR", "server-key.pem")
	RootClientCertFile   = configFile("CERT_DIR", "root-client.pem")
	RootClientKeyFile    = configFile("CERT_DIR", "root-client-key.pem")
	NobodyClientCertFile = configFile("CERT_DIR", "nobody-client.pem")
	NobodyClientKeyFile  = configFile("CERT_DIR", "nobody-client-key.pem")
	ACLModelFile         = configFile("ACL_DIR", "model.conf")
	ACLPolicyFile        = configFile("ACL_DIR", "policy.csv")
)

func configFile(configDirEnvVar, filename string) string {
	if dir := os.Getenv(configDirEnvVar); dir != "" {
		return filepath.Join(dir, filename)
	}

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(cwd, filename)
}
