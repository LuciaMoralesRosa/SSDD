// Implementacion de despliegue en ssh de multiples nodos
//
// Unica funcion exportada :
//		func ExecMutipleHosts(cmd string,
//							  hosts []string,
//							  results chan<- string,
//							  privKeyFile string)
//

package despliegue

import (
	"bufio"
	"bytes"

	//"fmt"

	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func getHostKey(host string) ssh.PublicKey {
	// parse OpenSSH known_hosts file
	// ssh or use ssh-keyscan to get initial key
	file, err := os.Open(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var hostKey ssh.PublicKey
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) != 3 {
			continue
		}
		if strings.Contains(fields[0], host) {
			var err error
			hostKey, _, _, _, err = ssh.ParseAuthorizedKey(scanner.Bytes())
			if err != nil {
				log.Fatalf("error parsing %q: %v", fields[2], err)
			}
			break
		}
	}

	if hostKey == nil {
		log.Fatalf("no hostkey found for %s", host)
	}

	return hostKey
}

// Ejecutar comando ssh remoto y devolver salida
func executeCmd(hostname string, cmd string, session *ssh.Session) string {
	//fmt.Println("APRES SESSION")

	var stdoutBuf bytes.Buffer

	session.Stdout = &stdoutBuf
	session.Stderr = &stdoutBuf

	//fmt.Println("ANTES RUN", cmd)

	session.Run(cmd)

	//fmt.Println("TRAS RUN", cmd)

	return hostname + ": \n" + stdoutBuf.String()
}

func buildSSHConfig(signer ssh.Signer) *ssh.ClientConfig {

	return &ssh.ClientConfig{
		User: os.Getenv("LOGNAME"),
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(signer),
		},
		// verify host public key
		//HostKeyCallback: ssh.FixedHostKey(hostKey),
		// Non-production only
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		// optional tcp connect timeout
		Timeout: 5 * time.Second,
	}
}

// Get signer from one of private key files id_ed25519, id_ecdsa or id_rsa
func getSigner(pkeyFile string) (ssh.Signer, error) {
	var signer ssh.Signer

	//Read private key file for user
	pkey, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".ssh", pkeyFile))

	//fmt.Println("PrivKey: ", string(pkey))

	if err == nil {
		// Create the Signer for this private key.
		signer, err = ssh.ParsePrivateKey(pkey)
	}

	return signer, err

}

// Ejecutar comando ssh en un host con autentifiación de clave pública
func execOneHost(hostname string, results chan<- string, cmd string) {

	pkeyFiles := []string{"id_ed25519", "id_ecdsa", "id_rsa"}

	for _, pkeyOneFile := range pkeyFiles {
		signer, errSigner := getSigner(pkeyOneFile)
		if errSigner == nil {
			// ssh_config must have option "HashKnownHosts no" !!!!
			//hostKey := getHostKey(hostname)
			//config := buildSSHConfig(signer, hostKey)
			config := buildSSHConfig(signer)

			conn, errConn := ssh.Dial("tcp", hostname+":22", config)
			if errConn != nil {
				continue
			}

			//fmt.Printf("APRES CONN %#v\n", config)

			session, errSession := conn.NewSession()
			if errSession != nil {
				conn.Close()
				continue
			}

			// ejecuta comano con buena sesión ssh al host remoto
			results <- executeCmd(hostname, cmd, session)

			session.Close()
			conn.Close()

			return // Se ha ejecutado correctamente
		}
	}

	log.Fatalf("NO funciona conexión ssh con clave pública a %s", hostname)
}

// Ejecutar un mismo comando en múltiples hosts mediante ssh
func ExecMutipleHosts(cmd string,
	hosts []string,
	results chan<- string) {

	//results := make(chan string, 1000)

	for _, hostname := range hosts {
		go execOneHost(hostname, results, cmd)
	}
}
