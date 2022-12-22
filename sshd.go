package main

import (
	"flag"
	"fmt"
	gossh "golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"syscall"
	"unsafe"
	"strings"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
)

var release = "dev" // Set by build process


// Define flags
var (
	bindHost    = flag.String("l", ":2222", "Listen <host:port>")
	hostKeyFile = flag.String("k", "~/.ssh/id_ed25519", "SSH host key file")
	kubeConfigFile = flag.String("c", "./config", "the kubectl config file, usually located in ~/.kube/config")
	verbose     = flag.Bool("v", false, "Enable verbose logging")
)

func handleAuth(ctx ssh.Context, providedPassword string) bool {
	log.Printf("New connection from %s user %s password %s\n", ctx.RemoteAddr(), ctx.User(), providedPassword)

	// Run the 'kubectl get pods' command to get a list of currently running pods
	exec.Command("source", "~/.profile")
	cmd := exec.Command("kubectl", "get", "pods")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return false
	}

	// Split the output by newline characters to get a slice of strings
	lines := strings.Split(string(output), "\n")

	// Iterate through the slice of strings and check if 's' is in the list
	for _, line := range lines {
		cols := strings.Fields(line)
		if (cols[0] == ctx.User()) {
			return true
		}
	}

	// If 's' is not found in the list, return false
	return false

	// Nate Sales Virsh Edition
	// ----------------------------
	// files, err := filepath.Glob("/etc/libvirt/qemu/*.xml")
	// if err != nil {
	// 	log.Fatalf("Unable to parse qemu config file glob: %v\n", err)
	// }

	// for _, f := range files {
	// 	// Read libvirt XML file
	// 	xmlFile, err := os.Open(f)
	// 	if err != nil {
	// 		log.Printf("XML open error: %v\n", err)
	// 	}

	// 	// Parse libvirt XML file
	// 	byteValue, _ := ioutil.ReadAll(xmlFile)
	// 	var currentDomain domain
	// 	err = xml.Unmarshal(byteValue, &currentDomain)
	// 	if err != nil {
	// 		log.Println(err)
	// 		return false
	// 	}
	// 	_ = xmlFile.Close()

	// 	if *verbose {
	// 		fmt.Printf("Found VM %s password %s\n", currentDomain.Name, currentDomain.Password)
	// 	}

	// 	if currentDomain.Name == ctx.User() && currentDomain.Password == providedPassword {
	// 		return true // Allow access
	// 	}
	// }

	// return false // If there are no defined VMs, deny access
}


func handleSession(s ssh.Session) {
	// kubectl exec <container> -it -- /bin/bash
	cmd := exec.Command("kubectl", "exec", s.User(), "-it", "--", "/bin/bash")
	cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", *kubeConfigFile))
	ptyReq, winCh, isPty := s.Pty() // get SSH PTY information
	if isPty {
		cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
		f, _ := pty.Start(cmd)
		go func() {
			for win := range winCh {
				_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(win.Height), uint16(win.Width), 0, 0})))
			}
		}()
		go func() { // goroutine to handle
			_, err := io.Copy(f, s) // stdin
			if err != nil {
				log.Printf("kubectl f->s copy error: %v\n", err)
			}
		}()
		_, err := io.Copy(s, f) // stdout
		if err != nil {
			log.Printf("kubectl s->f copy error: %v\n", err)
		}

		err = cmd.Wait()
		if err != nil {
			log.Printf("kubectl wait error: %v\n", err)
		}
	} else {
		_, _ = io.WriteString(s, "No PTY requested.\n")
		_ = s.Exit(1)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage for kubectl-sshd (%s) https://github.com/callowaysutton/kubectl-sshd:\n", release)
		flag.PrintDefaults()
	}

	flag.Parse()

	pemBytes, err := ioutil.ReadFile(*hostKeyFile)
	if err != nil {
		log.Fatal(err)
	}

	signer, err := gossh.ParsePrivateKey(pemBytes)
	if err != nil {
		log.Fatal(err)
	}

	sshServer := &ssh.Server{
		Addr:            *bindHost,
		HostSigners:     []ssh.Signer{signer},
		Handler:         handleSession,
		PasswordHandler: handleAuth,
	}
	log.Printf("Starting kubectl-sshd on %s\n", *bindHost)
	log.Fatal(sshServer.ListenAndServe())
}
