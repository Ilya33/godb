package godb

import (
	"github.com/dimonrus/gohelp"
	"github.com/dimonrus/porterr"
	"reflect"
	"strings"
)

// DB model interface
type IModel interface {
	Table() string
	Columns() []string
	Values() []interface{}
	Load(q Queryer) porterr.IError
	Save(q Queryer) porterr.IError
	Delete(q Queryer) porterr.IError
}

// IModelCrud CRUD query interface
type IModelCrud interface {
	GetLoadQuery() string
	GetInsertQuery() string
	GetUpdateQuery() string
	GetDeleteQuery() string
}

// DB model interface
type ISoftModel interface {
	IModel
	SoftLoad(q Queryer) porterr.IError
	SoftDelete(q Queryer) porterr.IError
	SoftRecover(q Queryer) porterr.IError
}

// Model column in db
func ModelColumn(model IModel, field interface{}) string {
	columns := ModelColumns(model, field)
	if len(columns) > 0 {
		return columns[0]
	}
	return ""
}

// Model columns by fileds
func ModelColumns(model IModel, field ...interface{}) (columns []string) {
	if model == nil {
		return
	}
	ve := reflect.ValueOf(model).Elem()
	te := reflect.TypeOf(model).Elem()
	var tField ModelFiledTag
	for j := range field {
		cte := reflect.ValueOf(field[j])
		if cte.Kind() != reflect.Ptr {
			continue
		}
		for i := 0; i < ve.NumField(); i++ {
			if ve.Field(i).Addr().Pointer() == cte.Elem().Addr().Pointer() {
				tField = ParseModelFiledTag(te.Field(i).Tag.Get("db"))
				columns = append(columns, tField.Column)
			}
		}
	}
	return
}

// Model values by columns
func ModelValues(model IModel, columns ...string) (values []interface{}) {
	if model == nil {
		return nil
	}
	te := reflect.TypeOf(model).Elem()
	modelValues := model.Values()
	values = make([]interface{}, len(columns))
	var tField ModelFiledTag
	var j int
	for i := 0; i < len(modelValues); i++ {
		tField = ParseModelFiledTag(te.Field(i).Tag.Get("db"))
		if gohelp.ExistsInArray(tField.Column, columns) {
			values[j] = modelValues[i]
			j++
		}
	}
	values = values[:j]
	return
}

// Model update query
func ModelUpdateQuery(model IModel, condition *Condition, fields ...interface{}) (sql string, params []interface{}, e porterr.IError) {
	if model == nil {
		e = porterr.New(porterr.PortErrorArgument, "Model is nil, check your logic")
		return
	}
	if fields == nil {
		e = porterr.New(porterr.PortErrorArgument, "Fields is empty, nothing for update")
		return
	}
	params = make([]interface{}, 0)
	var columns []string
	var tField ModelFiledTag
	ve := reflect.ValueOf(model).Elem()
	te := reflect.TypeOf(model).Elem()
	for i := 0; i < ve.NumField(); i++ {
		for _, v := range fields {
			cte := reflect.ValueOf(v)
			if cte.Kind() != reflect.Ptr {
				e = porterr.New(porterr.PortErrorArgument, "Fields must be an interfaces")
				return
			}
			if ve.Field(i).Addr().Pointer() == cte.Elem().Addr().Pointer() {
				tField = ParseModelFiledTag(te.Field(i).Tag.Get("db"))
				columns = append(columns, tField.Column+" = ?")
				params = append(params, v)
			}
		}
	}
	if len(columns) > 0 {
		sql = "UPDATE " + model.Table() + " SET " + strings.Join(columns, ",")
		if !condition.IsEmpty() {
			sql += " WHERE " + condition.String() + ";"
			params = append(params, condition.GetArguments()...)
		} else {
			sql += ";"
		}
	} else {
		e = porterr.New(porterr.PortErrorArgument, "No columns found in model")
	}
	return sql, params, e
}

// Model delete query
func ModelDeleteQuery(model IModel, condition *Condition) (sql string, e porterr.IError) {
	if model == nil {
		e = porterr.New(porterr.PortErrorArgument, "Model is nil, check your logic")
		return
	}
	sql = "DELETE FROM " + model.Table()
	if !condition.IsEmpty() {
		sql += " WHERE " + condition.String() + ";"
	} else {
		sql += ";"
	}
	return sql, e
}

// Model insert query
func ModelInsertQuery(model IModel, fields ...interface{}) (sql string, columns []string, e porterr.IError) {
	if model == nil {
		e = porterr.New(porterr.PortErrorArgument, "Model is nil, check the logic")
		return
	}
	ve := reflect.ValueOf(model).Elem()
	te := reflect.TypeOf(model).Elem()
	var tField ModelFiledTag
	if len(fields) > 0 {
		for i := 0; i < ve.NumField(); i++ {
			for _, v := range fields {
				cte := reflect.ValueOf(v)
				if cte.Kind() != reflect.Ptr {
					e = porterr.New(porterr.PortErrorArgument, "Fields must be an interfaces")
					return
				}
				if ve.Field(i).Addr().Pointer() == cte.Elem().Addr().Pointer() {
					tField = ParseModelFiledTag(te.Field(i).Tag.Get("db"))
					columns = append(columns, tField.Column)
				}
			}
		}
	} else {
		for i := 0; i < ve.NumField(); i++ {
			tField = ParseModelFiledTag(te.Field(i).Tag.Get("db"))
			if !tField.IsIgnored && !tField.IsSequence && ve.Field(i).CanSet() {
				columns = append(columns, tField.Column)
			}
		}
	}
	if len(columns) > 0 {
		sql = "INSERT INTO " + model.Table() + " (" + strings.Join(columns, ",") + ") "
	} else {
		e = porterr.New(porterr.PortErrorArgument, "No columns found in model")
	}
	return sql, columns, e
}
