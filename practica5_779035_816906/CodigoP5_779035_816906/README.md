# <font color="orange">Práctica 5</font>
Sistemas Distribuidos 2024
---
---
En esta práctica se ha desplegado el algoritmo de Raft implementado en practicas
anteriores en Kubernetes haciendo uso de Docker para los contendedores.

### <font color="brown">Requisitos previos</font>
Se necesita un sistema Linux (nosotros empleamos Debian12) con 25GB de disco y
4GB de RAM. Será necesario tener instalado Docker en la máquina y será necesario
añadir los dos ficheros contenidos en la carpeta "Necesarios" en algún directorio
del PATH de la máquina. También es preciso tener Go instalado.

Se recomienda que el usuario que ejecute el programa tenga permisos sudo sin
contraseña para agilizar la puesta en marcha.

### <font color="brown">Ejecución</font>
Para poner todo en marcha solo se debe ejecutar el script "deploy.sh"
- ./deploy.sh