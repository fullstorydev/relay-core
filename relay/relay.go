package relay

import (
	"log"
	"os"
)

var logger = log.New(os.Stdout, "[relay] ", 0)
