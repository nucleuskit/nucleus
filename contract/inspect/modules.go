package inspect

import (
	"bufio"
	"os"
	"strings"
)

// readModules 读取go.mod
func readModules(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var modules []string
	scanner := bufio.NewScanner(file)
	inRequire := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "require (":
			inRequire = true
		case inRequire && line == ")":
			inRequire = false
		case strings.HasPrefix(line, "require "):
			modules = append(modules, strings.Fields(strings.TrimPrefix(line, "require "))[0])
		case inRequire && line != "":
			modules = append(modules, strings.Fields(line)[0])
		}
	}
	return modules
}
