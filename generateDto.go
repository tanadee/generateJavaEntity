package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"time"
	"tnd/work/generateJavaEntity/cascadeMapping"
)

func generateDto(table TableWithRelation) error {
	restServiceTemplate, err := createTemplate("Dto").Parse(`// generated at {{.Time}}
{{- with .Package}}
package {{.}}.restservice;
{{- end}}
public class {{.Table.TypeName}}Dto {
	{{- range .Table.BasicColumns}}
		{{- if index $.Table.PrimaryKeys .Name}}
		{{- else}}
	private {{.Type | javaType}} {{.Name | camelCase}};
		{{- end}}
	{{- end}}

	{{- range .Table.Relations}}
		{{- if call $.IsCascade $.Table.TableIdentity .TableIdentity}}
			{{- if .ToMany}}
	private List<{{.TypeName}}Dto> {{.FieldName | pluralName}};
			{{- else}}
	private {{.TypeName}}Dto {{.FieldName}};
			{{- end}}
		{{- else}}
			{{- if .OwnField}}
				{{- if .ToMany}}
	private List<IdWrapperDto> {{.FieldName | pluralName}};
				{{- else}}
	private IdWrapperDto {{.FieldName}};
				{{- end}}
			{{- end}}
		{{- end}}
	{{- end}}

	{{- range .Table.BasicColumns}}
		{{- if index $.Table.PrimaryKeys .Name}}
		{{- else}}
	public {{.Type | javaType}} get{{.Name | camelCase | firstToUpper}}() {
		return {{.Name | camelCase}};
	}
	public void set{{.Name | camelCase | firstToUpper}}({{.Type | javaType}} {{.Name | camelCase}}) {
		this.{{.Name | camelCase}} = {{.Name | camelCase}};
	}
		{{- end}}
	{{- end}}

	{{- range .Table.Relations}}
		{{- if call $.IsCascade $.Table.TableIdentity .TableIdentity}}
			{{- if .ToMany}}
	public List<{{.TypeName}}Dto> get{{.FieldName | pluralName | firstToUpper}}() {
		return {{.FieldName | pluralName}};
	}
	public void set{{.FieldName | pluralName | firstToUpper}}(List<{{.TypeName}}Dto> {{.FieldName | pluralName}}) {
		this.{{.FieldName | pluralName}} = {{.FieldName | pluralName}};
	}
			{{- else}}
	public {{.TypeName}}Dto get{{.FieldName | firstToUpper}}() {
		return {{.FieldName}};
	}
	public void set{{.FieldName | firstToUpper}}({{.TypeName}}Dto {{.FieldName}}) {
		this.{{.FieldName}} = {{.FieldName}};
	}
			{{- end}}
		{{- else}}
			{{- if .OwnField}}
				{{- if .ToMany}}
	public List<IdWrapperDto> get{{.FieldName | pluralName | firstToUpper}}() {
		return {{.FieldName | pluralName}};
	}
	public void set{{.FieldName | pluralName | firstToUpper}}(List<IdWrapperDto> {{.FieldName | pluralName}}) {
		this.{{.FieldName | pluralName}} = {{.FieldName | pluralName}};
	}
				{{- else}}	
	public IdWrapperDto get{{.FieldName | firstToUpper}}() {
		return {{.FieldName}};
	}
	public void set{{.FieldName | firstToUpper}}(IdWrapperDto {{.FieldName}}) {
		this.{{.FieldName}} = {{.FieldName}};
	}
				{{- end}}
			{{- end}}
		{{- end}}
	{{- end}}
}
`)
	if err != nil {
		return fmt.Errorf("template parse: %w", err)
	}
	var resultWriter io.Writer
	if *generateFile {
		fileName := "generated/restservice/" + table.TypeName + "Dto.java"
		os.MkdirAll("generated/restservice", 0755)
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
	err = restServiceTemplate.Execute(resultWriter, map[string]interface{}{
		"Time":      time.Now(),
		"Table":     table,
		"Package":   *packageName,
		"IsCascade": cascadeMapping.IsCascadeRelation,
	})
	return err
}
