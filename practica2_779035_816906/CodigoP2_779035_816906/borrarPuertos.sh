#!/bin/bash

# Rango de puertos a cerrar
puertoInicial=31110
puertoFinal=31119

echo "Cerrando puertos del rango $puertoInicial al $puertoFinal..."

# Iterar sobre cada puerto en el rango
for puerto in $(seq $puertoInicial $puertoFinal); do
    # Buscar procesos asociados al puerto
    pid=$(lsof -ti :$puerto)
    
    if [ -n "$pid" ]; then
        echo "Matando proceso en el puerto $puerto con PID $pid"
        kill -9 $pid
        echo "Puerto $puerto liberado."
    else
        echo "No se encontró ningún proceso en el puerto $puerto."
    fi
done

echo "Borrado de procesos en los puertos completado"
