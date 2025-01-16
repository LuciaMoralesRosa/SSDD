# <font color="orange">Práctica 1</font>
Sistemas Distribuidos 2024
---
---
En esta práctica se ha implementado una barrera distribuida y cuatro variantes
de la arquitectura Cliente-Servidor.

### <font color="brown">Barrera</font>
Para ejecutar la barrera distribuida de numProc procesos:
- Definir en "endpoints.txt" las máquinas que se quieren ejecutar.
- Ejecutar desde la carpeta "barrier" de cada una de ellas:
    - go run barrier.go endpoints.txt <font color="brown">numProc</font>
        - <font color="brown">numProc</font> = [1, numProcesos]

### <font color="brown"> Cliente - Servidor </font>
En la carpeta "cmd" encontramos la subcarpeta "client" que contiene la
implementación de un cliente para una arquitectura cliente-servidor. En la
subcarpeta "server-draft" hay distintas implementaciones del servidor.

Para todos las ejecuciones, vamos a tomar la máquina *192.168.3.1* como cliente
y la máquina *192.168.3.2* como servidor (master en la master-worker) que 
escuchará por el puerto 31110.

#### <font color="orange">Servidor secuencial</font>
Ejecuta la tarea solicitada por el cliente de forma secuencial.
- Ejecutar en el cliente (*192.168.3.1*):
    - go run main.go <font color="maroon">ip:puerto (del server)</font>
    - go run main.go 192.168.3.2:31110
- Ejecutar en el servidor (*192.168.3.2*) desde la carpeta "serv-sec":
    - go run server_secuencial.go <font color="maroon">ip:puerto</font>
    - go run server_secuencial.go 192.168.3.2:31110

#### <font color="orange">Servidor concurrente</font>
Ejecuta varias tareas de forma concurrente lanzando una goroutine por tarea.
- Ejecutar en el cliente (*192.168.3.1*):
    - go run main.go <font color="maroon">ip:puerto (del server)</font>
    - go run main.go 192.168.3.2:31110
- Ejecutar en el servidor (*192.168.3.2*) desde la carpeta "serv-conc":
    - go run server_concurrente.go <font color="maroon">ip:puerto</font>
    - go run server_concurrente.go 192.168.3.2:31110

#### <font color="orange">Servidor con pool</font>
El servidor lanza varios workers que trabajarán de forma concurrente.
- Ejecutar en el cliente (*192.168.3.1*):
    - go run main.go <font color="maroon">ip:puerto (del server)</font>
    - go run main.go 192.168.3.2:31110
- Ejecutar en el servidor (*192.168.3.2*) desde la carpeta "serv-pool":
    - go run server_pool.go <font color="maroon">ip:puerto</font>
    - go run server_pool.go 192.168.3.2:31110

#### <font color="orange">Master-Worker</font>
El master se ejecutará en la máquina 192.168.3.2 y lanzará workers en las 
máquinas especificadas en el fichero "maquinas.txt" de la carpeta "master".
- Ejecutar en el cliente (*192.168.3.1*):
    - go run main.go <font color="maroon">ip:puerto (del server)</font>
    - go run main.go 192.168.3.2:31110
- Ejecutar en el master (*192.168.3.2*):
    - ./encenderMaquinas maquinas.txt
    - go run server_pool.go <font color="maroon">ip:puerto</font>
    - go run master.go 192.168.3.2:31110


