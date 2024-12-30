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
 
 

### cd rapido a las carpetas

# cliente
cd /misc/alumnos/sd/sd2425/a816906/Practica1_G26/cmd/client

# serv-secuencial
cd /misc/alumnos/sd/sd2425/a816906/Practica1_G26/cmd/server-draft/serv-secuencial

# serv-concurrente
cd /misc/alumnos/sd/sd2425/a816906/Practica1_G26/cmd/server-draft/server-conc

# server-pool
cd /misc/alumnos/sd/sd2425/a816906/Practica1_G26/cmd/server-draft/serv-pool

# master
cd /misc/alumnos/sd/sd2425/a816906/Practica1_G26/cmd/server-draft/serv-master

# worker
cd /misc/alumnos/sd/sd2425/a816906/Practica1_G26/cmd/server-draft/serv-worker



 Aroa:
 
