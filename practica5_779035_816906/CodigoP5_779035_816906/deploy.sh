export PATH=$PATH:$(pwd) # Lo ponemos en el path


# REINICIO CLUSTER -------------------------------------------------------------
echo "Cerrando el cluster (si es que habia alguno abierto)"
./cerrarCluster.sh

echo "\n\nCreando cluster con kind-with-registry.sh"
./kind-with-registry.sh


# SERVIDOR ---------------------------------------------------------------------
echo "\n\n\nCOMPILACION DEL SERVIDOR"
echo "\n\tEliminando fichero Dockerfile de la carpeta Docker/servidor..."
rm  ./Docker/servidor/srvraft >/dev/null 2>&1

cd ./raft # Acceso a la carpeta raft

echo "\n\tCreando fichero Dockerfile de la carpeta Docker/servidor..."
CGO_ENABLED=0 go build -o ../Docker/servidor/srvraft cmd/srvraft/main.go
cd ./../Docker/servidor # Acceso a la carpeta del servidor

echo "\n\tConstruccion del docker Servidor a partir del Dockerfile creado"
docker build . -t localhost:5001/srvraft:latest
echo "\n\tEvio de la imagen docker creada"
docker push localhost:5001/srvraft:latest

echo "\nCOMPILACION DEL SERVIDOR FINALIZADA"

echo "\n-----------------------------------"
cd ./../.. # Acceso a carpeta raiz


# CLIENTE ----------------------------------------------------------------------
echo "\nCOMPILACION DEL CLIENTE"
echo "\n\tEliminando ficheros Dockerfile de la carpeta Docker/cliente..."
rm  ./Docker/cliente/cltraft >/dev/null 2>&1

cd ./raft # Acceso a la carpeta raft

echo "\n\tCreando fichero Dockerfile de la carpeta Docker/cliente..."
CGO_ENABLED=0 go build -o ../Docker/cliente/cltraft pkg/cltraft/cltraft.go
cd ./../Docker/cliente # Acceso a la carpeta del cliente

echo "\n\tConstruccion del docker Cliente a partir del Dockerfile creado"
docker build . -t localhost:5001/cltraft:latest
echo "\n\tEvio de la imagen docker creada"
docker push localhost:5001/cltraft:latest

echo "\nCOMPILACION DEL CLIENTE FINALIZADA"

echo "\n-----------------------------------"
cd ./../.. # Acceso a carpeta raiz


# EJECUCION DE LOS PODS --------------------------------------------------------
echo "\n\n Ejecutando los pods...."
./expods.sh


