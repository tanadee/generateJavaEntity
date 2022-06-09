package cascadeMapping

import (
	"strings"
	"tnd/work/generateJavaEntity/tableDefinition"
)

func IsCascadeRelation(from tableDefinition.TableIdentity, to tableDefinition.TableIdentity) bool {
	result := from.Schema == to.Schema && strings.HasPrefix(to.Name, from.Name+"_") && !strings.Contains(to.Name[len(from.Name):], "_FORM")
	return result
}
