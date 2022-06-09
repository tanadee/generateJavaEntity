package tableDefinition

type ColumnDef struct {
	Position int
	Name     string
	Type     string
	Length   int
	Scale    int
}

type ForeignKey struct {
	Constname  string
	From       TableIdentity
	To         TableIdentity
	FkColnames string
	PkColnames string
}

type TableIdentity struct {
	Schema string
	Name   string
}

type TableDef struct {
	TableIdentity
	Columns     []ColumnDef
	ForeignKeys []ForeignKey
	PrimaryKeys map[string]bool
}
