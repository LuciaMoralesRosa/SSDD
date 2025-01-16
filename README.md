# <font color="orange">SISTEMAS DISTRIBUIDOS</font>
<font color="brown">Prácticas de Sistemas Distribuidos 2024 - Ingeniería Informática Unizar</font>
---
---
## Contenido
 Este repositorio contiene las prácticas de la asignatura de Sistemas 
 Distribuidos de Unizar.

### Práctica 1
- Implementación de una barrera distribuida.
- Implementación de las siguientes arquitecturas cliente-servidor:
    - Ejecución secuencual.
    - Ejecución concurrente.
    - Ejecución con pool de gorountines.
    - Arquitectura Master-Worker.

### Práctica 2
- Implementación del algoritmo de Ricart-Agrawala generalizado para el 
problema de lectores y escritores. (Habría sido mejor emplear los relojes de
la librería vclock)

### Practica 3
- Primera versión del algoritmo de consenso distribuido Raft para un problema
de clave valor. No se tiene en cuenta la consistencia del log de los candidatos
si se replican las entradas en las maquinas de estados.
- No se ha implementado la barrera distribuida.

### Práctica 4
- Versión final del algoritmo de Raft donde se valoran los casos no contemplados
en la versión anterior.

### Práctica 5
- Despligue del algoritmo de Raft implementado anteriormente en Kubernetes,
haciendo uso de Docker para la creación de contenedores.

----
## Comandos utiles empleados
### Acceso maquinas cluster

**Copiar carpeta en central:**
scp -r <nombreCarpeta> <font color="maroon">aNIP</font>@central.cps.unizar.es:*rutaDestino1*
- rutaDestino1 -> ~/Practicas/SSDD

**Copiar archivos de central a las maquinas:**
scp -r <nombreCarpeta> <font color="maroon">aNIP</font>@central.cps.unizar.es:*rutaDestino2*
- rutaDestino2 -> /home/$USER


**Acceso a central:**
ssh -Y <font color="maroon">aNIP</font>@central.cps.unizar.es

**Acceso a la maquina X:**
ssh <font color="maroon">aNIP</font>@192.168.3.X


### Creacion ssh para no necesiar contraseña
#### De local a central
- ssh-keygen -t rsa -b 4096 -> crear clave publica
- ssh-copy-id <font color="maroon">aNIP</font>@central.cps.unizar.es -> enviar clave publica a central
- ssh <font color="maroon">aNIP</font>@central.cps.unizar.es -> acceso a central

#### De central (o maquina remota) a maquina remota X
- ssh-keygen -t rsa -b 4096 -> crear clave publica
- ssh-copy-id <font color="maroon">aNIP</font>@192.168.3.X -> enviar clave publica a central
- ssh <font color="maroon">aNIP</font>@192.168.3.X -> acceso a central