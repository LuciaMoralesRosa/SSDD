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
