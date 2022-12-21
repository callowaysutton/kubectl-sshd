# kubectl-sshd

SSH daemon to interact with kubernete pod serial consoles 

### Installation

Do not recommend using in a production environment yet. Compile and use the service file.

### Usage

```
Usage for kubectl-sshd (dev) https://github.com/callowaysutton/kubectl-sshd:
  -k string
        SSH host key file (default "~/.ssh/id_ed25519")
  -l string
        Listen <host:port> (default ":2222")
  -p string
        Path to virsh binary (default "virsh")
  -v    Enable verbose logging
```
