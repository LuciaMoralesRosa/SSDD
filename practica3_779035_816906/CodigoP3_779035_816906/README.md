# <font color="orange">Práctica 3</font>
Sistemas Distribuidos 2024
---
---
En esta práctica se ha implementado una primera versión del algoritmo de consenso
distribuido Raft. En esta primera versión no se tiene en cuenta la consistencia 
del registro de los candidatos ni se replican las operaciones en las máquinas de
estado de los servidores.

### <font color="brown">Ejecución con el cliente implementado</font>
Para ejecutar el cliente se debe hacer de forma manual. Para la ejecución del 
cliente se hará uso de la máquina "192.168.3.1" y los servidores se ejecutarán 
en las máquinas 192.168.3.2, 192.168.3.3 y 192.168.3.4.
- Ejecución del cliente desde la carpeta "pkg/cltraft":
	- go run main.go
- Ejecución de los servidores desde la carpeta "cmd/srvraft":
	- go run main.go i 192.168.3.2:31110 192.168.3.3:31111 192.168.3.4:31112
	- Donde i es el identificador del proceso


### <font color="brown">Ejecución de las pruebas</font>
Desde la máquina 192.168.3.1, se debe ejecutar el comando:
- ./ej.sh