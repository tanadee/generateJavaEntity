package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

func generateJavaRestService(table TableWithRelation) error {
	if table.IdType == "" {
		return nil
	}
	restServiceTemplate, err := createTemplate("RestService").Parse(`// generated at {{.Time}}
{{- with .Package}}
package {{.}}.restservice;
{{- end}}

import java.util.List;
import java.util.Map;
import java.util.Optional;

import org.modelmapper.ModelMapper;
import org.modelmapper.convention.MatchingStrategies;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import com.fasterxml.jackson.databind.JsonNode;

{{- with .Package}}
import {{.}}.entity.{{$.Table.TypeName}};
import {{.}}.repository.{{$.Table.TypeName}}Repository;
{{- end}}
import th.go.cgd.ip.shared.api.RequestContext;
import th.go.cgd.ip.shared.api.RestfulOperationFlags;
import th.go.cgd.ip.shared.api.RestfulService;

@Service
public class {{.Table.TypeName}}RestService implements RestfulService<{{.Table.IdType}}, {{.Table.TypeName}}Dto, Map<String,List<String>>> {
	
	{{.Table.TypeName}}Repository {{.Table.TypeName | firstToLower}}Repository;
	DtoToEntityMapper dtoToEntityMapper;

	@Autowired
	public {{.Table.TypeName}}RestService({{.Table.TypeName}}Repository {{.Table.TypeName | firstToLower}}Repository, DtoToEntityMapper dtoToEntityMapper) {
		this.{{.Table.TypeName | firstToLower}}Repository = {{.Table.TypeName | firstToLower}}Repository;
		this.dtoToEntityMapper = dtoToEntityMapper;
	}
	
	@Override
	public RestfulOperationFlags getSupportedRestfulOperation() {
		RestfulOperationFlags flags = new RestfulOperationFlags();
		flags.set(RestfulOperationFlags.DELETE_BY_ID);
		flags.set(RestfulOperationFlags.PUT_BY_ID);
		flags.set(RestfulOperationFlags.POST);
		flags.set(RestfulOperationFlags.PATCH_BY_ID);
		return flags;
	}

	@Override
	public String getBaseResourceRelativePath() {
		return "/{{.Table.TypeName | camelToHyphen}}";
	}

	@Override
	@Transactional
	public void putById(RequestContext<Map<String, List<String>>> context, {{.Table.IdType}} id, {{.Table.TypeName}}Dto model) throws Exception {
		{{.Table.TypeName}} entity = {{.Table.TypeName | firstToLower}}Repository.findById(id).orElseThrow(RuntimeException::new);
		dtoToEntityPipeEntityManagerPersist(model, entity);
	}
	
	@Override
	@Transactional
	public Long post(RequestContext<Map<String, List<String>>> context, {{.Table.TypeName}}Dto model) throws Exception {
		{{.Table.TypeName}} entity = new {{.Table.TypeName}}();
		dtoToEntityPipeEntityManagerPersist(model, entity);
		return entity.get{{.Table.IdField | firstToUpper}}();
	}
	
	@Override
	@Transactional
	public void patchById(RequestContext<Map<String, List<String>>> context, {{.Table.IdType}} id, JsonNode node) throws Exception {
		{{.Table.TypeName}} entity = {{.Table.TypeName | firstToLower}}Repository.findById(id).orElseThrow(RuntimeException::new);
		jsonNodeToEntityPipeEntityManagerPersist(node, entity);
	}
	
	@Override
	@Transactional
	public boolean deleteById(RequestContext<Map<String, List<String>>> context, {{.Table.IdType}} id) throws Exception {
		Optional<{{.Table.TypeName}}> oEntity = {{.Table.TypeName | firstToLower}}Repository.findById(id);
		if (oEntity.isPresent()) {
			{{.Table.TypeName | firstToLower}}Repository.delete(oEntity.get());
			return true;
		}
		return false;
	}

	void dtoToEntityPipeEntityManagerPersist({{.Table.TypeName}}Dto dto, {{.Table.TypeName}} entity) throws Exception {
		dtoToEntity(dto, entity);
		{{.Table.TypeName | firstToLower}}Repository.save(entity);
		
		//side effect
		sideEffect(entity);
	}

	void jsonNodeToEntityPipeEntityManagerPersist(JsonNode node, {{.Table.TypeName}} entity) throws Exception {
		jsonToEntity(node, entity);
		{{.Table.TypeName | firstToLower}}Repository.save(entity);
		
		//side effect
		sideEffect(entity);
	}

	public void sideEffect({{.Table.TypeName}} entity) throws Exception {}

	public void dtoToEntity({{.Table.TypeName}}Dto dto, {{.Table.TypeName}} entity) {
		dtoToEntityMapper.mapDtoToEntity(dto, entity);
	}
	public void jsonToEntity(JsonNode node, {{.Table.TypeName}} entity) {
		dtoToEntityMapper.mapJsonToEntity(node, entity);
	}
}
`)
	if err != nil {
		return fmt.Errorf("template parse: %w", err)
	}
	var resultWriter io.Writer
	if *generateFile {
		fileName := "generated/restservice/" + table.TypeName + "RestService.java"
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
		"Time":    time.Now(),
		"Table":   table,
		"Package": *packageName,
	})
	return err
}
