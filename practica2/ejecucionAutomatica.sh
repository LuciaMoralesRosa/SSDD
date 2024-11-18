
scp -r "/home/$USER/Practicas/SSDD/practica2" "$USER@192.168.3.1:/home/$USER"

direcciones=("192.168.3.2" "192.168.3.3" "192.168.3.4")

# Copiar en todas las maquinas
for ip in "${direcciones[@]}"
do
    ssh "$USER@ip" "cd /home/$USER && rm -r practica2"

    scp -r "/home/$USER/Practicas/SSDD/practica2" "$USER@ip:/home/$USER"

    ssh "$USER@ip" "cd /home/$USER/practica2/cmd/escritor/ && go build"
    ssh "$USER@ip" "cd /home/$USER/practica2/cmd/lector/ && go build"
done