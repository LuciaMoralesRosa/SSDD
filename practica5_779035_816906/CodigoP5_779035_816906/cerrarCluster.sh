export PATH=$PATH:$(pwd) # Lo ponemos en el path

kind delete cluster

docker stop kind-worker
docker stop kind-worker2
docker stop kind-worker3
docker stop kind-worker4
docker stop kind-control-plane
docker stop kind-registry

docker rm kind-registry
docker rm kind-control-plane
docker rm kind-worker
docker rm kind-worker2
docker rm kind-worker3
docker rm kind-worker4
