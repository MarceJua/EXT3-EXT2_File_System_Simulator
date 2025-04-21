package analyzer

import (
	// Importa el paquete "errors" para manejar errores
	"fmt"     // Importa el paquete "fmt" para formatear e imprimir texto
	"strings" // Importa el paquete "strings" para manipulación de cadenas

	commands "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/commands" // Importa el paquete "commands" que contiene las funciones para analizar comandos
)

// splitCommand divide la entrada respetando cadenas entre comillas
// splitCommand divide la entrada respetando cadenas entre comillas
func splitCommand(input string) []string {
	var tokens []string
	var currentToken strings.Builder
	inQuotes := false

	for i := 0; i < len(input); i++ {
		char := input[i]

		switch char {
		case '"':
			inQuotes = !inQuotes
			// No agregamos las comillas al token, pero las respetamos en el proceso
		case ' ':
			if inQuotes {
				currentToken.WriteByte(char)
			} else if currentToken.Len() > 0 {
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
			}
		default:
			currentToken.WriteByte(char)
		}
	}

	if currentToken.Len() > 0 {
		tokens = append(tokens, currentToken.String())
	}

	return tokens
}

// Analyzer analiza el comando de entrada y ejecuta la acción correspondiente
func Analyzer(input string) (string, error) {
	// Eliminar espacios en blanco al inicio y final
	input = strings.TrimSpace(input)
	if input == "" {
		return "", nil // Input vacío no es un error, simplemente no hay nada que procesar
	}

	// Dividir la entrada en tokens respetando comillas
	tokens := splitCommand(input)
	if len(tokens) == 0 {
		return "", nil // Si no hay tokens válidos, devolvemos vacío
	}

	// Convertir el comando a minúsculas para hacerlo case-insensitive
	command := strings.ToLower(tokens[0])

	// Ejecutar el comando correspondiente
	switch command {
	case "mkdisk":
		return commands.ParseMkdisk(tokens[1:])
	case "rmdisk":
		return commands.ParseRmdisk(tokens[1:])
	case "fdisk":
		return commands.ParseFdisk(tokens[1:])
	case "mount":
		return commands.ParseMount(tokens[1:])
	case "mounted": // Asumo que esto es un comando personalizado para listar particiones montadas
		return commands.ParseMounted(tokens[1:])
	case "mkfs":
		return commands.ParseMkfs(tokens[1:])
	case "rep":
		return commands.ParseRep(tokens[1:])
	case "mkdir":
		return commands.ParseMkdir(tokens[1:])
	case "login":
		return commands.ParseLogin(tokens[1:])
	case "logout":
		return commands.ParseLogout(tokens[1:])
	case "mkgrp":
		return commands.ParseMkgrp(tokens[1:])
	case "mkfile":
		return commands.ParseMkfile(tokens[1:])
	case "rmgrp":
		return commands.ParseRmgrp(tokens[1:])
	case "mkusr":
		return commands.ParseMkusr(tokens[1:])
	case "rmusr":
		return commands.ParseRmusr(tokens[1:])
	case "chgrp":
		return commands.ParseChgrp(tokens[1:])
	case "cat":
		return commands.ParseCat(tokens[1:])
	default:
		return "", fmt.Errorf("comando desconocido: %s", command)
	}
}
