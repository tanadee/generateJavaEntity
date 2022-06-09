package main

import (
	"database/sql"
	"fmt"
	"strings"
	"tnd/work/generateJavaEntity/tableDefinition"
)

type ColumnDef = tableDefinition.ColumnDef
type ForeignKey = tableDefinition.ForeignKey
type TableDef = tableDefinition.TableDef
type TableIdentity = tableDefinition.TableIdentity

func listColumns(db *sql.DB, table TableIdentity) ([]ColumnDef, error) {
	st, err := db.Prepare(`select colno, colname, typename, length, scale from syscat.columns where tabschema = ? and tabname = ? order by colno`)
	if err != nil {
		return nil, fmt.Errorf("db prepare: %w", err)
	}
	defer st.Close()
	rs, err := st.Query(table.Schema, table.Name)
	if err != nil {
		return nil, fmt.Errorf("rs query: %w", err)
	}
	defer rs.Close()
	var defs []ColumnDef
	for rs.Next() {
		var def ColumnDef
		err := rs.Scan(&def.Position, &def.Name, &def.Type, &def.Length, &def.Scale)
		if err != nil {
			return nil, fmt.Errorf("rs scan: %w", err)
		}
		defs = append(defs, def)
	}
	return defs, nil
}

func listPK(db *sql.DB, table TableIdentity) (map[string]bool, error) {
	st, err := db.Prepare(`select key.colname from syscat.tabconst const, syscat.keycoluse key where const.type='P' and const.constname=key.constname and key.tabschema=? AND key.TABNAME=?`)
	if err != nil {
		return nil, fmt.Errorf("db prepare: %w", err)
	}
	defer st.Close()
	rs, err := st.Query(table.Schema, table.Name)
	if err != nil {
		return nil, fmt.Errorf("rs query: %w", err)
	}
	defer rs.Close()
	pks := make(map[string]bool)
	for rs.Next() {
		var pk string
		err := rs.Scan(&pk)
		if err != nil {
			return nil, fmt.Errorf("rs scan: %w", err)
		}
		pks[pk] = true
	}
	return pks, nil
}

func listFk(db *sql.DB, where string, argument ...interface{}) ([]ForeignKey, error) {
	st, err := db.Prepare(`select constname, tabschema, tabname, reftabschema, reftabname, fk_colnames, pk_colnames from syscat.references where ` + where)
	if err != nil {
		return nil, fmt.Errorf("db prepare: %w", err)
	}
	defer st.Close()
	rs, err := st.Query(argument...)
	if err != nil {
		return nil, fmt.Errorf("rs query: %w", err)
	}
	defer rs.Close()
	var result []ForeignKey
	for rs.Next() {
		var fk ForeignKey
		err := rs.Scan(&fk.Constname, &fk.From.Schema, &fk.From.Name, &fk.To.Schema, &fk.To.Name, &fk.FkColnames, &fk.PkColnames)
		if err != nil {
			return nil, fmt.Errorf("rs scan: %w", err)
		}
		fk.FkColnames = strings.TrimSpace(fk.FkColnames)
		fk.PkColnames = strings.TrimSpace(fk.PkColnames)
		fk.From.Schema = strings.TrimSpace(fk.From.Schema)
		fk.To.Schema = strings.TrimSpace(fk.To.Schema)
		fk.From.Name = strings.TrimSpace(fk.From.Name)
		fk.To.Name = strings.TrimSpace(fk.To.Name)
		result = append(result, fk)
	}
	return result, nil
}

func listFkFromTable(db *sql.DB, table TableIdentity) ([]ForeignKey, error) {
	return listFk(db, "tabschema = ? and tabname = ?", table.Schema, table.Name)
}

func ListFkToTable(db *sql.DB, table TableIdentity) ([]ForeignKey, error) {
	return listFk(db, "reftabschema = ? and reftabname = ?", table.Schema, table.Name)
}

func GetTableDef(db *sql.DB, table TableIdentity) (TableDef, error) {
	var tableDef TableDef
	var err error
	tableDef.TableIdentity = table
	if tableDef.Columns, err = listColumns(db, table); err != nil {
		return tableDef, fmt.Errorf("list column error: %w", err)
	}
	if tableDef.PrimaryKeys, err = listPK(db, table); err != nil {
		return tableDef, fmt.Errorf("list pk error: %w", err)
	}
	if tableDef.ForeignKeys, err = listFkFromTable(db, table); err != nil {
		return tableDef, fmt.Errorf("list pk error: %w", err)
	}
	return tableDef, nil
}
