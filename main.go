package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"text/template"
	"time"
	"tnd/work/generateJavaEntity/cascadeMapping"

	_ "github.com/ibmdb/go_ibm_db"
)

var (
	host     = flag.String("host", "ipdbs41", "hostname of database")
	port     = flag.String("port", "50100", "port of database")
	database = flag.String("database", "ipdb", "database name of database")
	uid      = flag.String("uid", "onldb", "username of database")
	pwd      = flag.String("pwd", "onldb", "password of username of database")
	schema   = flag.String("schema", "ONLDB", "schema of database")
	table    = flag.String("table", "COMPENSATION", "table of schema of database")
)

var (
	packageName  = flag.String("package", "th.go.cgd.ip.io", "package name of generated entity. Empty string omit package statement")
	generateFile = flag.Bool("file", true, "generate file instead of stdout")
)

const javaEntityTemplateText = `// generated at {{.Time}}
{{- with .Package}}
package {{.}}.entity;
{{end}}
import java.io.Serializable;

import java.math.BigDecimal;

import javax.persistence.CascadeType;
import javax.persistence.Column;
import javax.persistence.Entity;
import javax.persistence.FetchType;
import javax.persistence.GeneratedValue;
import javax.persistence.GenerationType;
import javax.persistence.Id;
import javax.persistence.JoinColumn;
import javax.persistence.ManyToMany;
import javax.persistence.ManyToOne;
import javax.persistence.MapsId;
import javax.persistence.OneToMany;
import javax.persistence.OneToOne;
import javax.persistence.SequenceGenerator;
import javax.persistence.Table;
import javax.persistence.Transient;

import java.time.LocalDate;
import java.time.LocalDateTime;

import java.util.List;
import java.util.ArrayList;

import th.go.cgd.ip.shared.entity.AuditData;

@Entity
@Table(schema = "{{.Table.Schema}}", name="{{.Table.Name}}")
public class {{.Table.Name | camelCase | firstToUpper }} {{if .Table.Audited}}extends AuditData {{end}} implements Serializable {

	private static final long serialVersionUID = 1L;

	{{range .Table.BasicColumns}}
		{{- if isId .Name -}}
	@Id
			{{- if $.Table.NoSeq}}
			{{- else}}
				{{- with sequenceName .Name}}
	@GeneratedValue(strategy = GenerationType.SEQUENCE, generator = "{{.}}")
	@SequenceGenerator(schema="{{$.Table.Schema}}", name="{{.}}", sequenceName="{{.}}", initialValue = 1, allocationSize = 1)
				{{- end}}
			{{- end}}
		{{- end}}
	@Column(name="{{.Name}}"{{. | colSpec}}) // Database's type is {{.Type}}
	private {{.Type | javaType}} {{.Name | camelCase}};
	{{end}}
	{{- range .Table.Relations}}
		{{range .Annotation}}
	{{.}}
		{{- end}}
		{{- if .ToMany}}
	private List<{{.TypeName}}> {{.FieldName | pluralName}} = new ArrayList<>();
		{{- else}}
	private {{.TypeName}} {{.FieldName}};
		{{- end}}
	{{end}}

	{{- range .Table.BasicColumns}}	
	public {{.Type | javaType}} get{{.Name | camelCase | firstToUpper}}() {
		return {{.Name | camelCase}};
	}
	public void set{{.Name | camelCase | firstToUpper}}({{.Type | javaType}} {{.Name | camelCase}}) {
		this.{{.Name | camelCase}} = {{.Name | camelCase}};
	}
	{{end}}
	{{- range .Table.Relations}}
		{{- if .ToMany}}
	public List<{{.TypeName}}> get{{.FieldName | pluralName | firstToUpper}}() {
		return {{.FieldName | pluralName}};
	}
	public void set{{.FieldName | pluralName | firstToUpper}}(List<{{.TypeName}}> {{.FieldName | pluralName}}) {
		this.{{.FieldName | pluralName}} = {{.FieldName | pluralName}};
	}
		{{- else}}
	public {{.TypeName}} get{{.FieldName | firstToUpper}}() {
		return {{.FieldName}};
	}
	public void set{{.FieldName | firstToUpper}}({{.TypeName}} {{.FieldName}}) {
		this.{{.FieldName}} = {{.FieldName}};
	}
		{{- end}}
	{{end}}
}
`

type javaEntityTemplateContext struct {
	Table      TableWithRelation
	EntityName string
	Time       time.Time
	Package    string
}

type ConnectParam struct {
	Host     string
	Port     string
	Database string
	UID      string
	PWD      string
}

func connectDB(param ConnectParam) (*sql.DB, error) {
	temp, err := template.New("con").Parse("HOSTNAME={{.Host}};DATABASE={{.Database}};PORT={{.Port}};UID={{.UID}};PWD={{.PWD}};AUTHENTICATION=SERVER")
	if err != nil {
		return nil, fmt.Errorf("template parse: %w", err)
	}
	conBuffer := &bytes.Buffer{}
	err = temp.Execute(conBuffer, param)
	if err != nil {
		return nil, fmt.Errorf("template execute: %w", err)
	}
	db, err := sql.Open("go_ibm_db", conBuffer.String())
	if err != nil {
		return nil, fmt.Errorf("sql open: %w", err)
	}
	return db, nil
}

func isTableManyToManyRelation(table TableDef) bool {
	if len(table.Columns) == 2 {
		if len(table.ForeignKeys) == 2 {
			if len(table.PrimaryKeys) == 2 {
				return true
			}
		}
	}
	return false
}

func camelCase(colName string) string {
	tokens := strings.Split(colName, "_")
	for i := range tokens {
		tokens[i] = strings.ToLower(tokens[i])
		if i > 0 {
			tokens[i] = strings.Title(tokens[i])
		}
	}
	return strings.Join(tokens, "")
}

func columnTypeToJavaType(typeName string) string {
	switch typeName {
	case "DATE":
		return "LocalDate"
	case "VARCHAR":
		return "String"
	case "BIGINT":
		return "Long"
	case "INTEGER":
		return "Integer"
	case "BOOLEAN":
		return "Boolean"
	case "VARGRAPHIC":
		return "String"
	case "TIMESTAMP":
		return "LocalDateTime"
	case "DECIMAL":
		return "BigDecimal"
	case "CLOB":
		return "String"
	}
	return typeName
}

func camelToHyphen(val string) string {
	var result []rune
	for i, r := range val {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result = append(result, '-')
			}
			result = append(result, r-'A'+'a')
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func pluralName(name string) string {
	if len(name) >= 2 {
		if name[len(name)-1] == 'y' {
			switch name[len(name)-2] {
			case 'a', 'e', 'i', 'o', 'u':
				return name + "s"
			default:
				return name[:len(name)-1] + "ies"
			}
		}
	}
	for _, suf := range []string{"s", "ss", "z", "ch", "sh", "x"} {
		if strings.HasSuffix(name, suf) {
			return name + "es"
		}
	}
	return name + "s"
}

func createTemplate(name string) *template.Template {
	return template.New(name).Funcs(map[string]interface{}{
		"javaType":     columnTypeToJavaType,
		"camelCase":    camelCase,
		"firstToUpper": strings.Title,
		"firstToLower": func(name string) string {
			return strings.ToLower(name[:1]) + name[1:]
		},
		"camelToHyphen": camelToHyphen,
		"toLower":       strings.ToLower,
		"pluralName":    pluralName,
	})
}

func generateJavaEntityByDefinition(table TableWithRelation) error {
	javaEntityTemplate, err := createTemplate("entity").Funcs(map[string]interface{}{
		"isId": func(colName string) bool {
			return table.PrimaryKeys[colName]
		},
		"sequenceName": func(colName string) string {
			if len(table.PrimaryKeys) == 1 {
				// if colName == "ID" || colName == table.Name+"_ID" {
				fmt.Println(table.Name + "_SEQ")
				return table.Name + "_SEQ"
				// }
			}
			return ""
		},
		"colSpec": func(col ColumnDef) string {
			switch col.Type {
			case "VARGRAPHIC":
				return fmt.Sprint(`, columnDefinition="VARGRAPHIC(`, col.Length, `)"`)
			case "VARCHAR":
				return fmt.Sprint(`, length=`, col.Length)
			}
			return ``
		},
	}).Parse(javaEntityTemplateText)
	if err != nil {
		return fmt.Errorf("text template parse: %w", err)
	}
	var entityName string = table.TypeName
	var resultWriter io.Writer
	if *generateFile {
		fileName := "generated/entity/" + entityName + ".java"
		os.MkdirAll("generated/entity", 0755)
		file, err := os.Create(fileName)
		if err != nil {
			return fmt.Errorf("os create %v: %w", fileName, err)
		}
		defer file.Close()
		resultWriter = file
	} else {
		buffer := new(bytes.Buffer)
		defer func() {
			log.Println(buffer.String())
		}()
		resultWriter = buffer
	}
	err = javaEntityTemplate.Execute(resultWriter, javaEntityTemplateContext{
		Table:   table,
		Time:    time.Now(),
		Package: *packageName,
	})
	if err != nil {
		return fmt.Errorf("template ExecuteL: %w", err)
	}
	return nil
}

func generateJavaRepository(table TableWithRelation) error {
	repoTemplate, err := template.New("repo").Parse(`// generated at {{.Time}}
{{- with .Package}}
package {{.}}.repository;
{{end}}
 
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.JpaSpecificationExecutor;
{{- with .Package}}

import {{.}}.entity.{{$.EntityTypeName}};
{{- end}}

public interface {{.EntityTypeName}}Repository extends JpaRepository<{{.EntityTypeName}},{{.PrimaryKeyTypeName}}>, JpaSpecificationExecutor<{{.EntityTypeName}}> {

}
`)
	if err != nil {
		return fmt.Errorf("template parse: %w", err)
	}
	var primaryKeyTypeName string
	for _, col := range table.BasicColumns {
		if table.PrimaryKeys[col.Name] {
			primaryKeyTypeName = columnTypeToJavaType(col.Type)
		}
	}
	if primaryKeyTypeName == "" {
		return nil
	}
	var resultWriter io.Writer
	if *generateFile {
		fileName := "generated/repository/" + table.TypeName + "Repository.java"
		os.MkdirAll("generated/repository", 0755)
		file, err := os.Create(fileName)
		if err != nil {
			return fmt.Errorf("os create %v: %w", fileName, err)
		}
		defer file.Close()
		resultWriter = file
	} else {
		buffer := new(bytes.Buffer)
		defer func() {
			log.Println(buffer.String())
		}()
		resultWriter = buffer
	}
	err = repoTemplate.Execute(resultWriter, map[string]interface{}{
		"Time":               time.Now(),
		"Package":            *packageName,
		"EntityTypeName":     table.TypeName,
		"PrimaryKeyTypeName": primaryKeyTypeName,
	})
	if err != nil {
		return fmt.Errorf("template execute: %w", err)
	}
	return nil
}

type ExtraRelation struct {
	TableIdentity
	Annotation []string
	ToMany     bool
	OwnField   bool
	TypeName   string
	FieldName  string
	MappedBy   string
}

type TableWithRelation struct {
	TableIdentity
	TypeName     string
	IdType       string
	IdField      string
	Audited      bool
	PrimaryKeys  map[string]bool
	NoSeq        bool
	BasicColumns []ColumnDef
	Relations    []ExtraRelation
}

func GetTableRelation(db *sql.DB, table TableDef) (TableWithRelation, error) {
	var result TableWithRelation
	result.TableIdentity = table.TableIdentity
	result.TypeName = strings.Title(camelCase(result.Name))
	result.PrimaryKeys = table.PrimaryKeys
	fks, err := ListFkToTable(db, table.TableIdentity)
	if err != nil {
		return result, err
	}
	for _, fk := range fks {
		fkTable, err := GetTableDef(db, fk.From)
		if err != nil {
			return result, fmt.Errorf("get table def %v: %w", fk.From, err)
		}
		if isTableManyToManyRelation(fkTable) {
			for _, otherFk := range fkTable.ForeignKeys {
				if otherFk != fk {
					if strings.HasPrefix(fkTable.Name, table.Name) {
						result.Relations = append(result.Relations, ExtraRelation{
							Annotation: []string{
								`@ManyToMany(fetch=FetchType.LAZY)`,
								`@JoinTable(name="` + fkTable.Name + `", schema="` + fkTable.Schema + `",`,
								`	joinColumns={@JoinColumn(name="` + fk.FkColnames + `")},`,
								`	inverseJoinColumns={@JoinColumn(name="` + otherFk.FkColnames + `")}`,
								`)`,
							},
							ToMany:        true,
							OwnField:      true,
							TableIdentity: otherFk.To,
							TypeName:      strings.Title(camelCase(otherFk.To.Name)),
							FieldName:     camelCase(fkTable.Name[len(table.Name)+1:]),
						})
					} else if strings.HasPrefix(fkTable.Name, otherFk.To.Name) {
						result.Relations = append(result.Relations, ExtraRelation{
							Annotation: []string{
								`@ManyToMany(fetch=FetchType.LAZY, mappedBy="` + camelCase(fkTable.Name[len(otherFk.To.Name)+1:]) + `")`,
							},
							ToMany:        true,
							OwnField:      false,
							MappedBy:      camelCase(fkTable.Name[len(otherFk.To.Name)+1:]),
							TableIdentity: otherFk.To,
							TypeName:      strings.Title(camelCase(otherFk.To.Name)),
							FieldName:     camelCase(otherFk.To.Name),
						})
					} else {
						fmt.Println("Many To Many Can't determine which Table is owner", fkTable.Schema+"."+fkTable.Name)
					}
				}
			}
		} else {
			if fkTable.PrimaryKeys[fk.FkColnames] {
				result.Relations = append(result.Relations, ExtraRelation{
					Annotation: []string{
						`@OneToOne(fetch=FetchType.LAZY, mappedBy="` + camelCase(table.Name) + `")`,
					},
					ToMany:        false,
					OwnField:      false,
					MappedBy:      camelCase(table.Name),
					TableIdentity: fkTable.TableIdentity,
					TypeName:      strings.Title(camelCase(fkTable.Name)),
					FieldName:     camelCase(fkTable.Name),
				})
			} else {
				otherColumnName := fk.FkColnames
				if strings.HasSuffix(otherColumnName, fk.PkColnames) {
					otherColumnName = otherColumnName[:len(otherColumnName)-len(fk.PkColnames)-1]
				}
				result.Relations = append(result.Relations, ExtraRelation{
					Annotation: []string{
						`@OneToMany(fetch=FetchType.LAZY, mappedBy="` + camelCase(otherColumnName) + `")`,
					},
					ToMany:        true,
					OwnField:      false,
					MappedBy:      camelCase(otherColumnName),
					TableIdentity: fkTable.TableIdentity,
					TypeName:      strings.Title(camelCase(fkTable.Name)),
					FieldName:     camelCase(fkTable.Name),
				})
			}
		}
	}
	fkColumn := make(map[string]ForeignKey)
	for _, fk := range table.ForeignKeys {
		fkColumn[fk.FkColnames] = fk
	}
	for _, col := range table.Columns {
		if fk, ok := fkColumn[col.Name]; ok {
			if table.PrimaryKeys[col.Name] {
				fieldName := col.Name
				if strings.HasSuffix(col.Name, "_"+fk.PkColnames) {
					fieldName = col.Name[:len(col.Name)-len(fk.PkColnames)-1]
				}
				result.Relations = append(result.Relations, ExtraRelation{
					Annotation: []string{
						`@OneToOne(fetch=FetchType.LAZY)`,
						`@MapsId`,
					},
					ToMany:        false,
					OwnField:      true,
					TableIdentity: fk.To,
					TypeName:      strings.Title(camelCase(fk.To.Name)),
					FieldName:     camelCase(fieldName),
				})
				result.BasicColumns = append(result.BasicColumns, col)
				if result.PrimaryKeys[col.Name] {
					result.IdType = columnTypeToJavaType(col.Type)
					result.IdField = camelCase(col.Name)
				}
				result.NoSeq = true
			} else {
				fieldName := col.Name
				if strings.HasSuffix(col.Name, "_"+fk.PkColnames) {
					fieldName = col.Name[:len(col.Name)-len(fk.PkColnames)-1]
				}
				result.Relations = append(result.Relations, ExtraRelation{
					Annotation: []string{
						`@ManyToOne(fetch=FetchType.LAZY)`,
						`@JoinColumn(name="` + col.Name + `")`,
					},
					ToMany:        false,
					OwnField:      true,
					TableIdentity: fk.To,
					TypeName:      strings.Title(camelCase(fk.To.Name)),
					FieldName:     camelCase(fieldName),
				})
			}
		} else {
			switch col.Name {
			case "MODIFIED_BY", "MODIFIED_DATE", "CREATED_BY", "CREATED_DATE":
				result.Audited = true
			default:
				result.BasicColumns = append(result.BasicColumns, col)
				if result.PrimaryKeys[col.Name] {
					result.IdType = columnTypeToJavaType(col.Type)
					result.IdField = camelCase(col.Name)
				}
			}
		}
	}

	for _, relation := range result.Relations {
		if cascadeMapping.IsCascadeRelation(result.TableIdentity, relation.TableIdentity) {
			mapping := relation.Annotation[0]
			if !strings.Contains(mapping, "ToOne") {
				mapping = mapping[:len(mapping)-1] + `, cascade=CascadeType.ALL, orphanRemoval=true)`
			} else {
				mapping = mapping[:len(mapping)-1] + `, cascade=CascadeType.ALL)`
			}
			relation.Annotation[0] = mapping
		}
	}
	return result, nil
}

func generate(connectParam ConnectParam, table TableIdentity, child map[string]bool) error {
	db, err := connectDB(connectParam)
	if err != nil {
		return fmt.Errorf("connectDB %w", err)
	}
	defer db.Close()
	tableDef, err := GetTableDef(db, table)
	if err != nil {
		return fmt.Errorf("get table def: %w", err)
	}
	tableWithRelation, err := GetTableRelation(db, tableDef)
	if err != nil {
		return fmt.Errorf("get table relation: %w", err)
	}

	var tableWithRelationList []TableWithRelation
	tableWithRelationList = append(tableWithRelationList, tableWithRelation)
	tableWithRelationMap := make(map[TableIdentity]bool)
	tableWithRelationMap[tableWithRelation.TableIdentity] = true
	for i := 0; i < len(tableWithRelationList); i++ {
		tableWithRelation := tableWithRelationList[i]
		for _, relation := range tableWithRelation.Relations {
			if cascadeMapping.IsCascadeRelation(tableWithRelation.TableIdentity, relation.TableIdentity) {
				if _, ok := tableWithRelationMap[relation.TableIdentity]; !ok {
					tableWithRelationMap[relation.TableIdentity] = true
					tableDef, err := GetTableDef(db, relation.TableIdentity)
					if err != nil {
						return fmt.Errorf("get table def: %w", err)
					}
					tableWithRelation, err := GetTableRelation(db, tableDef)
					if err != nil {
						return fmt.Errorf("get table relation: %w", err)
					}
					tableWithRelationList = append(tableWithRelationList, tableWithRelation)
				}
			}
		}
		err = generateJavaEntityByDefinition(tableWithRelation)
		if err != nil {
			return fmt.Errorf("generate by definition: %w", err)
		}
		err = generateJavaRepository(tableWithRelation)
		if err != nil {
			return fmt.Errorf("generateJavaRepository: %w", err)
		}
		err = generateDto(tableWithRelation)
		if err != nil {
			return fmt.Errorf("generateDto: %w", err)
		}
		err = generateJavaRestService(tableWithRelation)
		if err != nil {
			return fmt.Errorf("generateJavaRestService: %w", err)
		}
	}

	return nil
}

func run() error {
	flag.Parse()
	err := generate(ConnectParam{
		Host:     *host,
		Port:     *port,
		Database: *database,
		UID:      *uid,
		PWD:      *pwd,
	}, TableIdentity{
		Schema: *schema,
		Name:   *table,
	}, map[string]bool{
		"": true,
	})
	if err != nil {
		return fmt.Errorf("generate Java entity: %w", err)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Println(err)
	}
}
