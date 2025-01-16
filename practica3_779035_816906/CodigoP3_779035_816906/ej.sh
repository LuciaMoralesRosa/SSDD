#!/bin/bash

# -----------------------------------------------------------------------------#
# Ejecucion de los test de prueba de la practica3 para el algoritmo de Raft    #
# -----------------------------------------------------------------------------#

# Ejecución de los servidores en las maquinas remotas
ssh "$USER@$192.168.3.2" "clear && cd practica3/cmd/srvraft && go run main.go 0 192.168.3.2:31110 192.168.3.3:31111 192.168.3.4:31112" > /dev/null 2>&1 &
ssh "$USER@$192.168.3.3" "clear && cd practica3/cmd/srvraft && go run main.go 1 192.168.3.2:31110 192.168.3.3:31111 192.168.3.4:31112" > /dev/null 2>&1 &
ssh "$USER@$192.168.3.4" "clear && cd practica3/cmd/srvraft && go run main.go 2 192.168.3.2:31110 192.168.3.3:31111 192.168.3.4:31112" > /dev/null 2>&1 &

# Ejecución de los tests en la máquina actual
cd internal/testintegracionraft1
go get -u golang.org/x/crypto/ssh
go mod vendor
echo " ---------------- EJECUTANDO TESTS ---------------- "
go test -v ./...

exit 0


