package ssh

import (
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
	sshconfig "github.com/kevinburke/ssh_config"
)

// ResolveHost fills in missing SSH config fields from ~/.ssh/config.
func ResolveHost(host, user, identityFile string, port int) (string, string, string, int) {
	if user == "" {
		user = sshconfig.Get(host, "User")
	}
	if user == "" {
		user = os.Getenv("USER")
	}

	if identityFile == "" {
		identityFile = sshconfig.Get(host, "IdentityFile")
	}

	resolved := sshconfig.Get(host, "Hostname")
	if resolved != "" {
		host = resolved
	}

	if port == 0 {
		port = 22
	}

	return host, user, identityFile, port
}

// BuildAuthMethods returns SSH auth methods in priority order.
func BuildAuthMethods(useAgent bool, identityFile string) []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	if useAgent {
		if m := agentAuth(); m != nil {
			methods = append(methods, m)
		}
	}

	if identityFile != "" {
		if m := keyFileAuth(expandPath(identityFile)); m != nil {
			methods = append(methods, m)
		}
	}

	// Try common key paths as fallback
	home, _ := os.UserHomeDir()
	for _, name := range []string{"id_ed25519", "id_rsa", "id_ecdsa"} {
		p := filepath.Join(home, ".ssh", name)
		if expandPath(identityFile) == p {
			continue
		}
		if m := keyFileAuth(p); m != nil {
			methods = append(methods, m)
		}
	}

	return methods
}

func agentAuth() ssh.AuthMethod {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil
	}
	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers)
}

func keyFileAuth(path string) ssh.AuthMethod {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(signer)
}

// HostKeyCallback returns a known_hosts callback, or InsecureIgnoreHostKey if
// the known_hosts file doesn't exist.
func HostKeyCallback() ssh.HostKeyCallback {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".ssh", "known_hosts")
	cb, err := knownhosts.New(path)
	if err != nil {
		return ssh.InsecureIgnoreHostKey()
	}
	return cb
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}
