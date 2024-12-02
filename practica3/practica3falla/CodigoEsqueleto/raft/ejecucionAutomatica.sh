#!/bin/bash

puertoInicial=31110
puertoFinal=31119

# Liberar los puertos
for ((puerto = puertoInicial; puerto <= puertoFinal; puerto++)); do
    PIDS=$(lsof -ti :$puerto)
    if [ -n "$PIDS" ]; then
        for PID in $PIDS; do
            kill -9 $PID
        done
    fi
done

# Si no se pasa un argumento -> error
if [[ $# -ne 1 ]]; then
    echo "Uso: $0 <fichero_maquinas>"
    exit 1
fi

archivo="$1"
numeroNodos=0

# Comprobacion de que el fichero existe y no esta vacio
if [[ ! -f "$archivo" || ! -s "$archivo" ]]; then
    echo "No se encuentra el archivo o esta vacio"
    exit 1
fi

# Lista de nodos que se pasara como parametro al main
nodos=()

while IFS= read -r linea; do
    ((numeroNodos++))
    nodos+=("${linea}")
done < "$archivo"

for i in $(seq 0 $((numeroNodos - 1))); do
    comando="go run cmd/srvraft/main.go $i ${nodos[@]}"
    nohup $comando > "nodo_${i}.log" 2>&1 &
done