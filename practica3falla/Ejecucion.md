# Ejecucion Practica 3 - RAFT

## En local
### MANUAL
- En distintas terminales ejecutar desde la carpeta raft/cmd/srvraft:
- go run main.go 0 127.0.0.1:29001 127.0.0.1:29002 127.0.0.1:29003
- go run main.go 1 127.0.0.1:29001 127.0.0.1:29002 127.0.0.1:29003
- go run main.go 2 127.0.0.1:29001 127.0.0.1:29002 127.0.0.1:29003
### AUTOMATICO:
- Poner en maquinas.txt las direcciones locales <ip:puerto>
- Ejecutar ./ejecucionAutomatica.sh maquinas.txt

## En remoto
### MANUAL: 
- En distintas maquinas ejecutar: 
- go run main.go 0 192.163.3.2:31112 192.163.3.3:31113 192.163.3.4:31114
- go run main.go 1 192.163.3.2:31112 192.163.3.3:31113 192.163.3.4:31114
- go run main.go 2 192.163.3.2:31112 192.163.3.3:31113 192.163.3.4:31114
### AUTOMATICO
- Poner en maquinas.txt las direcciones remotas <ip:puerto>
- Ejecutar ./ejecucionAutomatica.sh maquinas.txt

### Pruebas
- Desde el directorio raft ejecutar:
- go get -u golang.org/x/crypto/ssh
- go mod vendor
- go test -v ./...
