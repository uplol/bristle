package bristle

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mailru/go-clickhouse"
	v1 "github.com/uplol/bristle/proto/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ClickhouseColumn struct {
	Name     string
	Position int
	Type     string
	Default  string
}

type ClickhouseTable struct {
	Name    FullTableName
	Columns map[string]*ClickhouseColumn

	writerIndex       int64
	writers           []*ClickhouseTableWriter
	config            *ClickhouseTableConfig
	cluster           *ClickhouseCluster
	cachedInsertQuery string
	cachedColumnNames []string
}

func newClickhouseTable(cluster *ClickhouseCluster, fullTableName FullTableName, columns map[string]*ClickhouseColumn, config ClickhouseTableConfig) (*ClickhouseTable, error) {
	table := &ClickhouseTable{
		Name:    fullTableName,
		Columns: columns,
		cluster: cluster,
		config:  &config,
		writers: make([]*ClickhouseTableWriter, 0),
	}

	if config.MaxBatchSize == 0 {
		config.MaxBatchSize = 10000
	}

	if config.FlushInterval == 0 {
		config.FlushInterval = 1000
	}

	if config.Writers == 0 {
		config.Writers = 1
	}

	if config.MessageInstancePoolSize == 0 {
		config.MessageInstancePoolSize = 32
	}

	if config.OnFull == "" {
		config.OnFull = "drop-oldest"
	}

	err := table.generateInsertQuery()
	if err != nil {
		return nil, err
	}

	return table, nil
}

func (t *ClickhouseTable) WriteBatch(batch [][]interface{}) error {
	result := atomic.AddInt64(&t.writerIndex, 1)
	t.writers[result%int64(len(t.writers))].buffer.WriteBatch(batch)
	return nil
}

// Describes the binding between a message type and its destination Clickhouse table.
type MessageTableBinding struct {
	Type         protoreflect.MessageType
	Table        *ClickhouseTable
	PrepareFunc  func(message protoreflect.Message) []interface{}
	InstancePool *MessageInstancePool
}

func (t *ClickhouseTable) generateInsertQuery() error {
	destinationName := string(t.Name)

	columnNames := make([]string, len(t.Columns))
	enclosedColumnNames := make([]string, len(t.Columns))
	for _, column := range t.Columns {
		columnNames[column.Position-1] = column.Name
		enclosedColumnNames[column.Position-1] = fmt.Sprintf(`"%s"`, column.Name)
	}

	varPlaceholders := []string{}
	for range columnNames {
		varPlaceholders = append(varPlaceholders, "?")
	}

	t.cachedInsertQuery = fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s);",
		destinationName,
		strings.Join(enclosedColumnNames, ","),
		strings.Join(varPlaceholders, ","),
	)
	t.cachedColumnNames = columnNames

	return nil
}

type preparedField struct {
	desc      protoreflect.FieldDescriptor
	canBeNull bool
	child     *preparedField

	defaultValue    interface{}
	defaultExpr     string
	isArray         bool
	isMapKey        bool
	isMapValue      bool
	isTimestamp     bool
	timestampFields [2]protoreflect.FieldDescriptor
}

func prepare(desc protoreflect.FieldDescriptor, column *ClickhouseColumn) preparedField {
	canBeNull := strings.HasPrefix(column.Type, "Nullable(")
	isArray := strings.HasPrefix(column.Type, "Array(")
	return preparedField{
		desc:        desc,
		canBeNull:   canBeNull,
		isArray:     isArray,
		defaultExpr: column.Default,
	}
}

func (t *ClickhouseTable) BindMessage(messageType protoreflect.MessageType, poolSize int) (*MessageTableBinding, error) {
	columnCount := len(t.Columns)
	columnFields := make([]preparedField, columnCount)

	fieldsIter := messageType.Descriptor().Fields()
	for i := 0; i < fieldsIter.Len(); i++ {
		field := fieldsIter.Get(i)

		columnName := string(field.Name())
		if proto.HasExtension(field.Options(), v1.E_BristleColumn) {
			columnName = proto.GetExtension(field.Options(), v1.E_BristleColumn).(string)
		}

		if field.IsMap() {
			keyColumn := t.Columns[string(field.Name())+".key"]
			if keyColumn == nil {
				return nil, fmt.Errorf("Failed to find key column for map '%v'", field.Name())
			}

			valueColumn := t.Columns[string(field.Name())+".value"]
			if valueColumn == nil {
				return nil, fmt.Errorf("Failed to find value column for map '%v'", field.Name())
			}

			columnFields[keyColumn.Position-1] = prepare(field, keyColumn)
			columnFields[keyColumn.Position-1].isMapKey = true
			columnFields[keyColumn.Position-1].defaultValue = []interface{}{}

			columnFields[valueColumn.Position-1] = prepare(field, valueColumn)
			columnFields[valueColumn.Position-1].isMapValue = true
			columnFields[valueColumn.Position-1].defaultValue = []interface{}{}
			continue
		}

		column, ok := t.Columns[columnName]
		if !ok {
			return nil, fmt.Errorf(
				"Failed to find column '%v' for field '%v'",
				columnName,
				field.Name(),
			)
		}

		columnFields[column.Position-1] = prepare(field, column)

		if field.Kind() == protoreflect.MessageKind {
			innerMessageFullName := field.Message().FullName()
			if innerMessageFullName == "google.protobuf.Timestamp" {
				columnFields[column.Position-1].isTimestamp = true
				columnFields[column.Position-1].timestampFields = [2]protoreflect.FieldDescriptor{
					field.Message().Fields().ByName("seconds"),
					field.Message().Fields().ByName("nanos"),
				}
			} else {
				return nil, fmt.Errorf("cannot handle arbitrary embedded message of type %v", innerMessageFullName)
			}
		}
	}

	prepare := func(message protoreflect.Message) []interface{} {
		result := make([]interface{}, columnCount)
		var ok bool
		for idx, field := range columnFields {
			result[idx], ok = getPreparedFieldValue(&field, message)
			if !ok {
				return nil
			}
		}
		return result
	}

	return &MessageTableBinding{
		Type:         messageType,
		Table:        t,
		PrepareFunc:  prepare,
		InstancePool: NewMessageInstancePool(messageType, poolSize),
	}, nil
}

func getPreparedFieldValue(field *preparedField, message protoreflect.Message) (interface{}, bool) {
	var err error
	var result interface{}

	if message.Has(field.desc) {
		if field.isTimestamp {
			seconds := message.Get(field.desc).Message().Get(field.timestampFields[0]).Int()
			nanos := message.Get(field.desc).Message().Get(field.timestampFields[1]).Int()
			result = time.Unix(seconds, nanos).UTC()
		} else if field.child != nil {
			return getPreparedFieldValue(field.child, message.Get(field.desc).Message())
		} else if field.isMapKey {
			fieldMap := message.Get(field.desc).Map()
			localResult := make([]interface{}, fieldMap.Len())
			idx := 0
			fieldMap.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
				localResult[idx] = key.Value().Interface()
				idx += 1
				return true
			})
			result = localResult
		} else if field.isMapValue {
			fieldMap := message.Get(field.desc).Map()
			localResult := make([]interface{}, fieldMap.Len())
			idx := 0
			fieldMap.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
				localResult[idx] = value.Interface()
				idx += 1
				return true
			})
			result = localResult
		} else {
			result = message.Get(field.desc).Interface()
		}
	} else {
		if field.defaultValue != nil {
			result = field.defaultValue
		} else if field.defaultExpr != "" {
			return field.defaultExpr, true
		} else if field.canBeNull {
			result = nil
		} else {
			// Missing field that cannot be null
			return nil, false
		}
	}

	if field.isArray && err == nil {
		return clickhouse.Array(result), true
	}

	return result, err == nil
}
