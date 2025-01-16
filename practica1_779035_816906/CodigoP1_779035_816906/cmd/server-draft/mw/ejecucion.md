--- LANZAMIENTO MANUAL DE WORKERS ----------------------------------------------

Desde la maquina 192.168.3.2
- ssh a816906@192.168.3.4 'cd practica1/cmd/server-draft/mw/worker && go run worker.go 192.168.3.4:31114'
- ssh a816906@192.168.3.3 'cd practica1/cmd/server-draft/mw/worker && go run worker.go 192.168.3.3:31113'
- go run master.go 192.168.3.2:31112 maquinas.txt

Desde el cliente:
- go run main.go 192.168.3.2:31112

--- LANZAMIENTO AUTOMATICO DE WORKERS CON SH -----------------------------------

Desde la maquina 192.168.3.2
- ./encenderMaquinas maquinas.txt a816906
- go run master.go 192.168.3.2:31112 maquinas.txt

Desde el cliente:
- go run main.go 192.168.3.2:31112



--- OTROS COMANDOS -------------------------------------------------------------
Matar procesos ssh
kill $(ps aux | grep 'ssh' | awk '{print $2}')

Matar procesos en los puertos 31113 de las maquinas
lsof -i :31113
kill -9 pid