# SSDD
 Practicas de Sistemas Distribuidos 2024 - Ingenieria Informatica Unizar

 ## Acceso maquinas cluster
 
 Copiar carpeta en central:
 scp -r Practica1_G26 a816906@central.cps.unizar.es:~/Practicas/SSDD

 Copiar archivos de central a las maquinas:
 

 Acceso a central:
 ssh -Y a816906@central.cps.unizar.es

 Acceso a la maquina:
 ssh a816906@192.168.3.X
 
 
 ## Creacion ssh

 ssh-keygen -t rsa -b 4096 -> crear clave publica
 ssh-copy-id a816906@central.cps.unizar.es -> enviar clave publica a la maquina remota
 ssh a816906@central.cps.unizar.es -> acceso a central
