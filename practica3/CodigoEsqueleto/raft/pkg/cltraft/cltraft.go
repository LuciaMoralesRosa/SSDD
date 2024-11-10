package cltraft

func obtenerMaquinas(fichero string) ([]string, error) {
	file, err := os.Open(fichero)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var maquinas []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			maquinas = append(maquinas, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return maquinas, nil
}


func main(){
	ficheroMaquinas := os.Args[1];
	maquinas := obtenerMaquinas(ficheroMaquinas)

	// Seleccionar lider
	lider := strings.Split(maquinas[0], ":")

	numeroDeMaquinas := 3

	for i := 0; i < numeroDeMaquinas; i++ {
		cliente, err := rpc.Dial("tcp", maquina[0])
		check.CheckError(err, "Fallo al establecer la conexion tcp")

		defer cliente.Close()

		if i % 2 == 0 {
			operacion := raft.TipoOperacion{Operacion: "escribir",
											Clave: "ClaveEscritura",
											Valor: "Escribir"}
		} else {
			operacion := raft.TipoOperacion{Operacion: "leer",
											Clave: "ClaveLectura",
											Valor: ""}
		}

		var respuesta raft.ResultadoRemoto
		err = cliente.Call("NodoRaft.SometerOperacionRaft", operacion, &respuesta)
		check.CheckError(err, "Error en la funcion SometerOperacionRaft")

		fmt.Println(respuesta)
		time.Sleep(1*time.Second)
	}
}