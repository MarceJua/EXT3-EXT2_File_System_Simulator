package structures

import (
	"fmt"
	"time"
)

// Crear users.txt en nuestro sistema de archivos
func (sb *SuperBlock) CreateUsersFile(path string) error {
	// ----------- Creamos / -----------
	rootInode := &Inode{
		I_uid:   1,
		I_gid:   1,
		I_size:  0,
		I_atime: float32(time.Now().Unix()),
		I_ctime: float32(time.Now().Unix()),
		I_mtime: float32(time.Now().Unix()),
		I_block: [15]int32{0, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, // Bloque 0
		I_type:  [1]byte{'0'},
		I_perm:  [3]byte{'7', '7', '7'},
	}
	err := rootInode.Serialize(path, int64(sb.S_inode_start)) // Inodo 0
	if err != nil {
		return fmt.Errorf("error al serializar inodo raíz: %v", err)
	}
	err = sb.UpdateBitmapInode(path, 0)
	if err != nil {
		return err
	}

	rootBlock := &FolderBlock{
		B_content: [4]FolderContent{
			{B_name: ToByte12("."), B_inodo: 0},
			{B_name: ToByte12(".."), B_inodo: 0},
			{B_name: ToByte12("users.txt"), B_inodo: 1},
			{B_name: ToByte12("-"), B_inodo: -1},
		},
	}
	err = rootBlock.Serialize(path, int64(sb.S_block_start)) // Bloque 0
	if err != nil {
		return fmt.Errorf("error al serializar bloque raíz: %v", err)
	}
	err = sb.UpdateBitmapBlock(path, 0)
	if err != nil {
		return err
	}

	// ----------- Creamos /users.txt -----------
	usersText := "1,G,root\n1,U,root,123\n"
	usersInode := &Inode{
		I_uid:   1,
		I_gid:   1,
		I_size:  int32(len(usersText)),
		I_atime: float32(time.Now().Unix()),
		I_ctime: float32(time.Now().Unix()),
		I_mtime: float32(time.Now().Unix()),
		I_block: [15]int32{1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, // Bloque 1
		I_type:  [1]byte{'1'},
		I_perm:  [3]byte{'7', '7', '7'},
	}
	err = usersInode.Serialize(path, int64(sb.S_inode_start+sb.S_inode_size)) // Inodo 1
	if err != nil {
		return fmt.Errorf("error al serializar inodo users.txt: %v", err)
	}
	err = sb.UpdateBitmapInode(path, 1)
	if err != nil {
		return err
	}

	usersBlock := &FileBlock{B_content: [64]byte{}}
	copy(usersBlock.B_content[:], usersText)
	err = usersBlock.Serialize(path, int64(sb.S_block_start+sb.S_block_size)) // Bloque 1
	if err != nil {
		return fmt.Errorf("error al serializar bloque users.txt: %v", err)
	}
	err = sb.UpdateBitmapBlock(path, 1)
	if err != nil {
		return err
	}

	fmt.Printf("DEBUG: users.txt escrito en bloque 1 con contenido: %s\n", usersText)
	sb.S_first_ino = 2 // Próximo inodo libre
	sb.S_first_blo = 2 // Próximo bloque libre

	return nil
}
