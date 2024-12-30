#/bin/bash

maquinas=("192.168.3.1" "192.168.3.2" "192.168.3.3" "192.168.3.4")
# maquinas=("192.168.3.3" "192.168.3.4" "192.168.3.6" "192.168.3.7")
# maquinas=("192.168.3.1" "192.168.3.3" "192.168.3.4" "192.168.3.6")

# Bucle para copiar y compilar el archivo en cada dirección IP
for ip in "${maquinas[@]}"
do
  # Borrar carpeta anterior
  ssh "$USER@$ip" "cd /home/$USER && rm -r practica2"

  # Copiar todos los ficheros de la práctica
  scp -r "/home/$USER/Practicas/SSDD/practica2" "$USER@$ip:/home/$USER"
done

exit 0
central:~/Practicas/SSDD/ 
