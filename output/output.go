package output

import (
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-io/bitrise/configs"
)

// Print ...
func Print(outModel interface{}, format string) {
	switch format {
	case configs.OutputFormatJSON:
		serBytes, err := json.Marshal(outModel)
		if err != nil {
			log.Errorf("[output.print] ERROR: %s", err)
			return
		}
		fmt.Printf("%s\n", serBytes)
	default:
		log.Errorf("[output.print] Invalid output format: %s", format)
	}
}
