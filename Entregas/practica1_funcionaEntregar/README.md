# SSDD
 Practicas de Sistemas Distribuidos 2024 - Ingenieria Informatica Unizar

 ## Acceso maquinas cluster
 
 Copiar archivos en el directorio:
 scp -r SSDD a816906@central.cps.unizar.es:/misc/alumnos/sd/sd2425/a816906

 Acceso a central:
 ssh -Y a816906@central.cps.unizar.es

 Acceso a la maquina:
 ssh a816906@192.168.3.X
 
 
 ## Creacion ssh:
 ssh-keygen -t rsa -b 4096 -> crear clave publica
 ssh-copy-id a816906@central.cps.unizar.es -> enviar clave publica a la maquina remota
 ssh a816906@central.cps.unizar.es -> acceso a central

 ssh-copy-id a816906@192.168.3.3
 ssh-copy-id a816906@192.168.3.4
 
 
## Ejecucion
Ejecutar Barrera:
- Poner todas las maquinas a ejecutar: go run barrier.go endpoints.txt <num> 

Ejecutar Servidores:
- Primero ejecutar cliente con: go run main.go <ip:puerto (del server)>
    - go run main.go 192.168.3.2:31110

- Ejecutar en maquina 192.168.3.2 el servidor x <ip:puerto (propios)>
    - go run server_secuencial.go 192.168.3.2:31110

Ejecutar master-worker:
 - Ejecutar en maquina 192.168.3.2 el master y lanzar las maquinas
    - ./encenderMaquinas maquinas.txt
    - go run master.go 192.168.3.2:31110

