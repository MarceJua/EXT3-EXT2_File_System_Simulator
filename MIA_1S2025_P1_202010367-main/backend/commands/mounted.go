package commands

import (
	"fmt"
	"strings"

	stores "github.com/MarceJua/MIA_1S2025_P1_202010367/backend/stores"
)

// MOUNTED estructura que representa el comando mounted (sin parámetros)
type MOUNTED struct{}

/*
   mounted
*/

func ParseMounted(tokens []string) (string, error) {
	if len(tokens) > 0 {
		return "", fmt.Errorf("el comando mounted no acepta parámetros, solo 'mounted'")
	}

	output, err := commandMounted()
	if err != nil {
		return "", err
	}
	return output, nil
}

func commandMounted() (string, error) {
	if len(stores.MountedPartitions) == 0 {
		return "MOUNTED: No hay particiones montadas actualmente", nil
	}

	var output strings.Builder
	output.WriteString("MOUNTED: Particiones montadas:\n")
	for id, path := range stores.MountedPartitions {
		output.WriteString(fmt.Sprintf("  ID: %s  Path: %s\n", id, path))
	}
	return output.String(), nil
}
