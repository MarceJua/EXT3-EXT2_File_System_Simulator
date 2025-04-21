package commands

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
	structures "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/structures"
)

// LOGIN estructura que representa el comando login con sus parámetros
type LOGIN struct {
	user string // Nombre del usuario
	pass string // Contraseña
	id   string // ID de la partición
}

// ParseLogin parsea los tokens del comando login
func ParseLogin(tokens []string) (string, error) {
	cmd := &LOGIN{}

	for _, token := range tokens {
		parts := strings.SplitN(token, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("formato inválido: %s", token)
		}
		key := strings.ToLower(parts[0])
		value := parts[1]

		switch key {
		case "-user":
			if value == "" {
				return "", errors.New("el usuario no puede estar vacío")
			}
			cmd.user = value
		case "-pass":
			if value == "" {
				return "", errors.New("la contraseña no puede estar vacía")
			}
			cmd.pass = value
		case "-id":
			if value == "" {
				return "", errors.New("el id no puede estar vacío")
			}
			cmd.id = value
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.user == "" || cmd.pass == "" || cmd.id == "" {
		return "", errors.New("faltan parámetros requeridos: -user, -pass, -id")
	}

	err := commandLogin(cmd)
	if err != nil {
		return "", fmt.Errorf("error al iniciar sesión: %v", err)
	}

	return fmt.Sprintf("LOGIN: Sesión iniciada como %s en %s", cmd.user, cmd.id), nil
}

func commandLogin(login *LOGIN) error {
	if stores.CurrentSession.ID != "" {
		return errors.New("ya hay una sesión activa, cierre la sesión actual primero")
	}

	partitionSuperblock, _, partitionPath, err := stores.GetMountedPartitionSuperblock(login.id)
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %w", err)
	}

	file, err := os.OpenFile(partitionPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir disco: %v", err)
	}
	defer file.Close()

	// Leer el inodo raíz (inodo 0)
	rootInode := &structures.Inode{}
	err = rootInode.Deserialize(partitionPath, int64(partitionSuperblock.S_inode_start))
	if err != nil {
		return fmt.Errorf("error al leer el inodo raíz: %w", err)
	}
	if rootInode.I_type[0] != '0' {
		return errors.New("el inodo raíz no es una carpeta válida")
	}

	// Buscar users.txt en el bloque raíz
	var usersInodeNum int32 = -1
	for _, blockNum := range rootInode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		folderBlock := &structures.FolderBlock{}
		err = folderBlock.Deserialize(partitionPath, int64(partitionSuperblock.S_block_start+blockNum*partitionSuperblock.S_block_size))
		if err != nil {
			return fmt.Errorf("error al leer el bloque %d de la raíz: %w", blockNum, err)
		}
		for _, content := range folderBlock.B_content {
			name := strings.Trim(string(content.B_name[:]), "\x00")
			if name == "users.txt" {
				usersInodeNum = content.B_inodo
				break
			}
		}
		if usersInodeNum != -1 {
			break
		}
	}
	if usersInodeNum == -1 {
		return errors.New("users.txt no encontrado en el directorio raíz")
	}

	// Leer el inodo de users.txt
	usersInode := &structures.Inode{}
	err = usersInode.Deserialize(partitionPath, int64(partitionSuperblock.S_inode_start+usersInodeNum*partitionSuperblock.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al leer el inodo de users.txt: %w", err)
	}
	if usersInode.I_type[0] != '1' {
		return errors.New("users.txt no es un archivo válido")
	}

	// Leer todos los bloques del inodo
	var content strings.Builder
	for i, blockNum := range usersInode.I_block[:12] {
		if blockNum == -1 {
			break
		}
		fileBlock := &structures.FileBlock{}
		err = fileBlock.Deserialize(partitionPath, int64(partitionSuperblock.S_block_start+blockNum*partitionSuperblock.S_block_size))
		if err != nil {
			return fmt.Errorf("error al leer el bloque %d de users.txt: %w", blockNum, err)
		}
		content.Write(bytes.Trim(fileBlock.B_content[:], "\x00"))
		fmt.Printf("DEBUG: Bloque %d leído: %s\n", i, strings.Trim(string(fileBlock.B_content[:]), "\x00"))
	}
	usersContent := strings.TrimSpace(content.String())
	fmt.Printf("DEBUG: Contenido de users.txt en login:\n%s\n", usersContent)

	lines := strings.Split(usersContent, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

		if len(parts) == 4 && parts[1] == "U" {
			username := parts[2]
			password := parts[3]
			if username == login.user && password == login.pass {
				stores.CurrentSession = stores.Session{
					ID:       login.id,
					Username: login.user,
					UID:      parts[0],
					GID:      parts[0],
				}
				return nil
			}
		}
	}

	return errors.New("usuario o contraseña incorrectos")
}
