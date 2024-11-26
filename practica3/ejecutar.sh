#!/bin/bash

endpoints=(
  ":31110" 
  ":31111" 
  ":31112"
  ":31113" 
  ":31114" 
  ":31115"
  ":31116" 
  ":31117" 
  ":31118" 
  ":31119")

# Clean ports
start_port=31110
end_port=31119
for ((port = start_port; port <= end_port; port++)); do
    PIDS=$(lsof -ti :$port)
    if [ -n "$PIDS" ]; then
        for PID in $PIDS; do
            kill -9 $PID
        done
    fi
done

# Read number of nodes
while true; do
  read -p "Enter the number of nodes between 2 and 10: " num

  # Check if the input is a number
  if ! [[ $num =~ ^[0-9]+$ ]]; then
    echo "Invalid input. Please enter a valid number."
    continue
  fi

  # Check if the number is within the range 2-10
  if ((num >= 2 && num <= 10)); then
    break
  else
    echo "Number is not in the range 2-10. Please try again."
  fi
done

# Print the value of num after the while loop
echo "Number of nodes: $num"

nodes=()
for ((i = 0; i < num; i++)); do
  nodes+=("${endpoints[i]}")
done

for i in $(seq 0 $((num-1))); do
  command="go run cmd/srvraft/main.go $i ${nodes[@]}"
  nohup $command 2>&1 &
done

#nohup comando-a-ejecutar > archivo-de-salida.log 2>&1 &
