package tools

import (
	// Blank import ensures that these repos are included in go.mod, so we can
	// build CLI tools from the ./vendor/ directory during `make install_deps`
	_ "github.com/golang/mock/mockgen"
)
