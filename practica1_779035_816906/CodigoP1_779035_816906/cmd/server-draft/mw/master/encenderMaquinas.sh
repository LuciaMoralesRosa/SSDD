#!/bin/bash

if [[ $# -ne 2 ]]; then
    echo "Uso: $0 <fichero_maquinas> <usuario_maquinas>"
    exit 1
fi

archivo="$1"
usuario="$2"

if [[ ! -f "$archivo" || ! -s "$archivo" ]]; then
    echo "No se encuentra el archivo o esta vacio"
    exit 1
fi

while IFS= read -r linea; do
    if [[ -n "$linea" ]]; then 
        ip=$(echo "$linea" | cut -d':' -f1)
        puerto=$(echo "$linea" | cut -d':' -f2)
        ssh "${usuario}@${ip}" "cd practica1/cmd/server-draft/mw/worker && go run worker.go $linea" &
    fi
done < "$archivo"
