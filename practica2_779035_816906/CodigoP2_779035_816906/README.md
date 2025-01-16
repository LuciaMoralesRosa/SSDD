# <font color="orange">Práctica 2</font>
Sistemas Distribuidos 2024
---
---
En esta práctica se ha implementado el algoritmo de Ricart-Agrawala generalizado
para el problema de lectores y escritores.

Para ejecutarlo se debe hacer lo siguiente:
- Definir en los ficheros "usuarios.txt" de las carpetas "cmd/escritor" y 
"cmd/lector" las máquinas y puertos que se van a ejecutar.
- Asegurarse de que el numero de procesos es el mismo definido en la constante
nProcesos del fichero "ra.go" de la carpeta "ra".
- Abrir las máquinas correspondientes a lo definido en el fichero "usuarios.txt".
- Acceder a la carpeta "cmd/escritor" en aquellas máquinas que se desee que 
sean escritores y a "cmd/lector" en las que se quiera que sean lectores.
- Ejecutar en cada máquina el siguiente comando:
    - go run main.go <font color="maroon">numProceso</font>
    - Donde <font color="maroon">numProceso</font> viene dado por su posicion en la lista de usuarios del
    fichero usuarios.txt