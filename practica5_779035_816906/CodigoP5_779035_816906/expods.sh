kubectl delete pod cliente
kubectl delete service raft-service
kubectl delete statefulset raft
echo "--------- Pequeña pausa de un segundo --------"
sleep 1
kubectl create -f pods.yaml
