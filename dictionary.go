package godb

import (
	"database/sql"
	_ "embed"
	"fmt"
	"github.com/dimonrus/gohelp"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

//go:embed dictionary_mapping.tmpl
var DefaultDictionaryTemplate string

type DictionaryModel struct {
	Id        int        `json:"id"`        // Идентификатор значения справочника
	Type      string     `json:"type"`      // Тип значения справочника
	Code      string     `json:"code"`      // Код значения справочника
	Label     *string    `json:"label"`     // Описание значения справочника
	CreatedAt time.Time  `json:"createdAt"` // Время создания записи
	UpdatedAt *time.Time `json:"updatedAt"` // Время обновления записи
	DeletedAt *time.Time `json:"deletedAt"` // Время удаления записи
}

// Model columns
func (m *DictionaryModel) Columns() []string {
	return []string{"id", "type", "code", "label", "created_at", "updated_at", "deleted_at"}
}

// Model values
func (m *DictionaryModel) Values() (values []interface{}) {
	return append(values, &m.Id, &m.Type, &m.Code, &m.Label, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt)
}

// Parse model column
func (m *DictionaryModel) parse(rows *sql.Rows) (*DictionaryModel, error) {
	err := rows.Scan(m.Values()...)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Search by filer
func (m *DictionaryModel) SearchDictionary(q Queryer) (*[]DictionaryModel, []int, error) {
	qb := NewQB().From("public.dictionary").
		Columns((&DictionaryModel{}).Columns()...).
		AddOrder("type", "created_at", "id")
	rows, err := q.Query(qb.String(), qb.GetArguments()...)

	entityIds := make([]int, 0)
	if err != nil {
		return nil, entityIds, err
	}
	defer rows.Close()
	var result []DictionaryModel
	for rows.Next() {
		row, err := (&DictionaryModel{}).parse(rows)
		if err != nil {
			return &result, entityIds, err
		}

		entityIds = append(entityIds, row.Id)
		result = append(result, *row)
	}
	return &result, entityIds, nil
}

// Create Table
func CreateDictionaryTable(q Queryer) error {
	query := `
CREATE TABLE IF NOT EXISTS dictionary
(
  id         INT PRIMARY KEY                                 NOT NULL,
  type       TEXT                                            NOT NULL,
  code       TEXT                                            NOT NULL,
  label      TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT localtimestamp NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE,
  deleted_at TIMESTAMP WITH TIME ZONE
);

COMMENT ON COLUMN dictionary.id IS 'Идентификатор значения справочника';
COMMENT ON COLUMN dictionary.type IS 'Тип значения справочника';
COMMENT ON COLUMN dictionary.code IS 'Код значения справочника';
COMMENT ON COLUMN dictionary.label IS 'Описание значения справочника';
COMMENT ON COLUMN dictionary.created_at IS 'Время создания записи';
COMMENT ON COLUMN dictionary.updated_at IS 'Время обновления записи';
COMMENT ON COLUMN dictionary.deleted_at IS 'Время удваления записи';

CREATE INDEX IF NOT EXISTS dictionary_type_idx ON dictionary (type);`

	_, err := q.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

// Create or update dictionary mapping
func GenerateDictionaryMapping(path string, q Queryer) error {
	dictionaries, _, err := (&DictionaryModel{}).SearchDictionary(q)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	paths := strings.Split(path, fmt.Sprintf("%c", os.PathSeparator))
	packageName := paths[len(paths)-2]
	tml := getDictionaryTemplate()
	err = tml.Execute(f, struct {
		Dictionaries []DictionaryModel
		Package      string
	}{
		Dictionaries: *dictionaries,
		Package:      packageName,
	})

	if err != nil {
		err = os.RemoveAll(path)
	}
	if err != nil {
		return err
	}

	cmd := exec.Command("go", "fmt", path)
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func getDictionaryTemplate() *template.Template {
	funcMap := template.FuncMap{
		"camelCase": func(str string) string {
			result, _ := gohelp.ToCamelCase(str, true)
			return result
		},
	}
	return template.Must(template.New("").Funcs(funcMap).Parse(DefaultDictionaryTemplate))
}
