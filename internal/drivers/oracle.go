package drivers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/reloquent/reloquent/internal/config"
)

// FindOracleJDBC scans ~/.reloquent/drivers/ for ojdbc*.jar files.
// Returns the path to the first found JAR, or an error if none found.
func FindOracleJDBC() (string, error) {
	driversDir := config.ExpandHome("~/.reloquent/drivers/")

	entries, err := os.ReadDir(driversDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("drivers directory not found: %s", driversDir)
		}
		return "", fmt.Errorf("reading drivers directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "ojdbc") && strings.HasSuffix(name, ".jar") {
			return filepath.Join(driversDir, name), nil
		}
	}

	return "", fmt.Errorf("no ojdbc*.jar found in %s", driversDir)
}

// OracleJDBCGuidance returns download instructions for the Oracle JDBC driver.
func OracleJDBCGuidance() string {
	return `# Oracle JDBC Driver Required
#
# The Oracle JDBC driver cannot be bundled due to licensing restrictions.
# To set up the driver:
#
# 1. Download ojdbc8.jar from:
#    https://www.oracle.com/database/technologies/appdev/jdbc-downloads.html
#
# 2. Place it in ~/.reloquent/drivers/
#    mkdir -p ~/.reloquent/drivers/
#    cp ojdbc8.jar ~/.reloquent/drivers/
#
# 3. When submitting the Spark job, include the JAR:
#    --jars ~/.reloquent/drivers/ojdbc8.jar
`
}
