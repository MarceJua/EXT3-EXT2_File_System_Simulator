package structures

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type SuperBlock struct {
	S_filesystem_type   int32
	S_inodes_count      int32
	S_blocks_count      int32
	S_free_inodes_count int32
	S_free_blocks_count int32
	S_mtime             float32
	S_umtime            float32
	S_mnt_count         int32
	S_magic             int32
	S_inode_size        int32
	S_block_size        int32
	S_first_ino         int32
	S_first_blo         int32
	S_bm_inode_start    int32
	S_bm_block_start    int32
	S_inode_start       int32
	S_block_start       int32
	// Total: 68 bytes
}

// Serialize escribe la estructura SuperBlock en un archivo binario en la posición especificada
func (sb *SuperBlock) Serialize(path string, offset int64) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Mover el puntero del archivo a la posición especificada
	_, err = file.Seek(offset, 0)
	if err != nil {
		return err
	}

	// Serializar la estructura SuperBlock directamente en el archivo
	err = binary.Write(file, binary.LittleEndian, sb)
	if err != nil {
		return err
	}

	return nil
}

// Deserialize lee la estructura SuperBlock desde un archivo binario en la posición especificada
func (sb *SuperBlock) Deserialize(path string, offset int64) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Mover el puntero del archivo a la posición especificada
	_, err = file.Seek(offset, 0)
	if err != nil {
		return err
	}

	// Obtener el tamaño de la estructura SuperBlock
	sbSize := binary.Size(sb)
	if sbSize <= 0 {
		return fmt.Errorf("invalid SuperBlock size: %d", sbSize)
	}

	// Leer solo la cantidad de bytes que corresponden al tamaño de la estructura SuperBlock
	buffer := make([]byte, sbSize)
	_, err = file.Read(buffer)
	if err != nil {
		return err
	}

	// Deserializar los bytes leídos en la estructura SuperBlock
	reader := bytes.NewReader(buffer)
	err = binary.Read(reader, binary.LittleEndian, sb)
	if err != nil {
		return err
	}

	return nil
}

// PrintSuperBlock imprime los valores de la estructura SuperBlock
func (sb *SuperBlock) Print() {
	// Convertir el tiempo de montaje a una fecha
	mountTime := time.Unix(int64(sb.S_mtime), 0)
	// Convertir el tiempo de desmontaje a una fecha
	unmountTime := time.Unix(int64(sb.S_umtime), 0)

	fmt.Printf("Filesystem Type: %d\n", sb.S_filesystem_type)
	fmt.Printf("Inodes Count: %d\n", sb.S_inodes_count)
	fmt.Printf("Blocks Count: %d\n", sb.S_blocks_count)
	fmt.Printf("Free Inodes Count: %d\n", sb.S_free_inodes_count)
	fmt.Printf("Free Blocks Count: %d\n", sb.S_free_blocks_count)
	fmt.Printf("Mount Time: %s\n", mountTime.Format(time.RFC3339))
	fmt.Printf("Unmount Time: %s\n", unmountTime.Format(time.RFC3339))
	fmt.Printf("Mount Count: %d\n", sb.S_mnt_count)
	fmt.Printf("Magic: %d\n", sb.S_magic)
	fmt.Printf("Inode Size: %d\n", sb.S_inode_size)
	fmt.Printf("Block Size: %d\n", sb.S_block_size)
	fmt.Printf("First Inode: %d\n", sb.S_first_ino)
	fmt.Printf("First Block: %d\n", sb.S_first_blo)
	fmt.Printf("Bitmap Inode Start: %d\n", sb.S_bm_inode_start)
	fmt.Printf("Bitmap Block Start: %d\n", sb.S_bm_block_start)
	fmt.Printf("Inode Start: %d\n", sb.S_inode_start)
	fmt.Printf("Block Start: %d\n", sb.S_block_start)
}

// Imprimir inodos
func (sb *SuperBlock) PrintInodes(path string) error {
	// Imprimir inodos
	fmt.Println("\nInodos\n----------------")
	// Iterar sobre cada inodo
	for i := int32(0); i < sb.S_inodes_count; i++ {
		inode := &Inode{}
		// Deserializar el inodo
		err := inode.Deserialize(path, int64(sb.S_inode_start+(i*sb.S_inode_size)))
		if err != nil {
			return err
		}
		// Imprimir el inodo
		fmt.Printf("\nInodo %d:\n", i)
		inode.Print()
	}

	return nil
}

// Impriir bloques
func (sb *SuperBlock) PrintBlocks(path string) error {
	// Imprimir bloques
	fmt.Println("\nBloques\n----------------")
	// Iterar sobre cada inodo
	for i := int32(0); i < sb.S_inodes_count; i++ {
		inode := &Inode{}
		// Deserializar el inodo
		err := inode.Deserialize(path, int64(sb.S_inode_start+(i*sb.S_inode_size)))
		if err != nil {
			return err
		}
		// Iterar sobre cada bloque del inodo (apuntadores)
		for _, blockIndex := range inode.I_block {
			// Si el bloque no existe, salir
			if blockIndex == -1 {
				continue
			}
			// Si el inodo es de tipo carpeta
			if inode.I_type[0] == '0' {
				block := &FolderBlock{}
				// Deserializar el bloque
				err := block.Deserialize(path, int64(sb.S_block_start+(blockIndex*sb.S_block_size))) // 64 porque es el tamaño de un bloque
				if err != nil {
					return err
				}
				// Imprimir el bloque
				fmt.Printf("\nBloque %d:\n", blockIndex)
				block.Print()
				continue

				// Si el inodo es de tipo archivo
			} else if inode.I_type[0] == '1' {
				block := &FileBlock{}
				// Deserializar el bloque
				err := block.Deserialize(path, int64(sb.S_block_start+(blockIndex*sb.S_block_size))) // 64 porque es el tamaño de un bloque
				if err != nil {
					return err
				}
				// Imprimir el bloque
				fmt.Printf("\nBloque %d:\n", blockIndex)
				block.Print()
				continue
			}

		}
	}

	return nil
}

// CreateFolder crea una carpeta en el sistema de archivos
func (sb *SuperBlock) CreateFolder(path string, parentsDir []string, destDir string) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir archivo: %v", err)
	}
	defer file.Close()

	// Empezar desde el inodo raíz (0)
	currentInode := &Inode{}
	err = currentInode.Deserialize(path, int64(sb.S_inode_start))
	if err != nil {
		return fmt.Errorf("error al leer inodo raíz: %v", err)
	}
	currentInodeNum := int32(0) // Raíz siempre es 0

	// Navegar o crear directorios padres
	for _, dir := range parentsDir {
		if dir == "" {
			continue
		}
		found := false
		for _, blockNum := range currentInode.I_block[:12] {
			if blockNum == -1 {
				break
			}
			folderBlock := &FolderBlock{}
			err = folderBlock.Deserialize(path, int64(sb.S_block_start+blockNum*sb.S_block_size))
			if err != nil {
				return fmt.Errorf("error al leer bloque %d: %v", blockNum, err)
			}
			for _, content := range folderBlock.B_content {
				name := strings.Trim(string(content.B_name[:]), "\x00")
				if name == dir {
					currentInodeNum = content.B_inodo
					err = currentInode.Deserialize(path, int64(sb.S_inode_start+content.B_inodo*sb.S_inode_size))
					if err != nil {
						return fmt.Errorf("error al leer inodo %d: %v", content.B_inodo, err)
					}
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			// Crear nuevo directorio padre
			newInodeNum, err := sb.FindFreeInode(path)
			if err != nil {
				return fmt.Errorf("error al encontrar inodo libre: %v", err)
			}
			newBlockNum, err := sb.FindFreeBlock(path)
			if err != nil {
				return fmt.Errorf("error al encontrar bloque libre: %v", err)
			}

			newInode := &Inode{
				I_uid:   1, // UID de root
				I_gid:   1, // GID de root
				I_size:  0,
				I_atime: float32(time.Now().Unix()),
				I_ctime: float32(time.Now().Unix()),
				I_mtime: float32(time.Now().Unix()),
				I_block: [15]int32{newBlockNum, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
				I_type:  [1]byte{'0'}, // Carpeta
				I_perm:  [3]byte{'7', '7', '7'},
			}
			err = newInode.Serialize(path, int64(sb.S_inode_start+newInodeNum*sb.S_inode_size))
			if err != nil {
				return fmt.Errorf("error al serializar nuevo inodo %d: %v", newInodeNum, err)
			}
			err = sb.UpdateBitmapInode(path, newInodeNum)
			if err != nil {
				return err
			}
			sb.S_free_inodes_count--

			newFolderBlock := &FolderBlock{
				B_content: [4]FolderContent{
					{B_name: ToByte12("."), B_inodo: newInodeNum},
					{B_name: ToByte12(".."), B_inodo: currentInodeNum},
					{B_name: ToByte12("-"), B_inodo: -1},
					{B_name: ToByte12("-"), B_inodo: -1},
				},
			}
			err = newFolderBlock.Serialize(path, int64(sb.S_block_start+newBlockNum*sb.S_block_size))
			if err != nil {
				return fmt.Errorf("error al serializar bloque %d: %v", newBlockNum, err)
			}
			err = sb.UpdateBitmapBlock(path, newBlockNum)
			if err != nil {
				return err
			}
			sb.S_free_blocks_count--

			// Vincular al padre
			for i, blockNum := range currentInode.I_block[:12] {
				if blockNum == -1 {
					currentInode.I_block[i] = newBlockNum
					break
				}
				folderBlock := &FolderBlock{}
				err = folderBlock.Deserialize(path, int64(sb.S_block_start+blockNum*sb.S_block_size))
				if err != nil {
					return err
				}
				for j, content := range folderBlock.B_content {
					if content.B_inodo == -1 {
						folderBlock.B_content[j] = FolderContent{B_name: ToByte12(dir), B_inodo: newInodeNum}
						err = folderBlock.Serialize(path, int64(sb.S_block_start+blockNum*sb.S_block_size))
						if err != nil {
							return err
						}
						break
					}
				}
			}
			err = currentInode.Serialize(path, int64(sb.S_inode_start+currentInodeNum*sb.S_inode_size))
			if err != nil {
				return fmt.Errorf("error al serializar inodo padre %d: %v", currentInodeNum, err)
			}
			currentInodeNum = newInodeNum
			err = currentInode.Deserialize(path, int64(sb.S_inode_start+currentInodeNum*sb.S_inode_size))
			if err != nil {
				return fmt.Errorf("error al leer nuevo inodo %d: %v", currentInodeNum, err)
			}
		}
	}

	// Crear el directorio final
	newInodeNum, err := sb.FindFreeInode(path)
	if err != nil {
		return fmt.Errorf("error al encontrar inodo libre para %s: %v", destDir, err)
	}
	newBlockNum, err := sb.FindFreeBlock(path)
	if err != nil {
		return fmt.Errorf("error al encontrar bloque libre para %s: %v", destDir, err)
	}

	newInode := &Inode{
		I_uid:   1,
		I_gid:   1,
		I_size:  0,
		I_atime: float32(time.Now().Unix()),
		I_ctime: float32(time.Now().Unix()),
		I_mtime: float32(time.Now().Unix()),
		I_block: [15]int32{newBlockNum, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		I_type:  [1]byte{'0'},
		I_perm:  [3]byte{'7', '7', '7'},
	}
	err = newInode.Serialize(path, int64(sb.S_inode_start+newInodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al serializar inodo %d: %v", newInodeNum, err)
	}
	err = sb.UpdateBitmapInode(path, newInodeNum)
	if err != nil {
		return err
	}
	sb.S_free_inodes_count--

	newFolderBlock := &FolderBlock{
		B_content: [4]FolderContent{
			{B_name: ToByte12("."), B_inodo: newInodeNum},
			{B_name: ToByte12(".."), B_inodo: currentInodeNum},
			{B_name: ToByte12("-"), B_inodo: -1},
			{B_name: ToByte12("-"), B_inodo: -1},
		},
	}
	err = newFolderBlock.Serialize(path, int64(sb.S_block_start+newBlockNum*sb.S_block_size))
	if err != nil {
		return fmt.Errorf("error al serializar bloque %d: %v", newBlockNum, err)
	}
	err = sb.UpdateBitmapBlock(path, newBlockNum)
	if err != nil {
		return err
	}
	sb.S_free_blocks_count--

	// Vincular al padre
	for i, blockNum := range currentInode.I_block[:12] {
		if blockNum == -1 {
			currentInode.I_block[i] = newBlockNum
			break
		}
		folderBlock := &FolderBlock{}
		err = folderBlock.Deserialize(path, int64(sb.S_block_start+blockNum*sb.S_block_size))
		if err != nil {
			return err
		}
		for j, content := range folderBlock.B_content {
			if content.B_inodo == -1 {
				folderBlock.B_content[j] = FolderContent{B_name: ToByte12(destDir), B_inodo: newInodeNum}
				err = folderBlock.Serialize(path, int64(sb.S_block_start+blockNum*sb.S_block_size))
				if err != nil {
					return err
				}
				break
			}
		}
	}
	err = currentInode.Serialize(path, int64(sb.S_inode_start+currentInodeNum*sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al serializar inodo padre %d: %v", currentInodeNum, err)
	}

	// Serializar el superbloque
	err = sb.Serialize(path, int64(sb.S_bm_inode_start)-int64(binary.Size(sb)))
	if err != nil {
		return fmt.Errorf("error al serializar superbloque: %v", err)
	}

	return nil
}

func (sb *SuperBlock) FindFreeInode(path string) (int32, error) {
	if sb.S_free_inodes_count <= 0 {
		return -1, errors.New("no hay inodos libres disponibles")
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return -1, err
	}
	defer file.Close()

	_, err = file.Seek(int64(sb.S_bm_inode_start), 0)
	if err != nil {
		return -1, err
	}

	bm := make([]byte, sb.S_inodes_count)
	_, err = file.Read(bm)
	if err != nil {
		return -1, err
	}

	for i := int32(0); i < sb.S_inodes_count; i++ {
		if bm[i] == '0' {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no se encontraron inodos libres, pero S_free_inodes_count es %d", sb.S_free_inodes_count)
}

// FindFreeBlock busca un bloque libre en el bitmap de bloques
func (sb *SuperBlock) FindFreeBlock(path string) (int32, error) {
	if sb.S_free_blocks_count <= 0 {
		return -1, errors.New("no hay bloques libres disponibles")
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return -1, err
	}
	defer file.Close()

	_, err = file.Seek(int64(sb.S_bm_block_start), 0)
	if err != nil {
		return -1, err
	}

	bm := make([]byte, sb.S_blocks_count)
	_, err = file.Read(bm)
	if err != nil {
		return -1, err
	}

	for i := int32(0); i < sb.S_blocks_count; i++ {
		if bm[i] == '0' {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no se encontraron bloques libres, pero S_free_blocks_count es %d", sb.S_free_blocks_count)
}

// toByte12 convierte un string a un array de 12 bytes
func ToByte12(name string) [12]byte {
	var b [12]byte
	copy(b[:], name)
	return b
}
