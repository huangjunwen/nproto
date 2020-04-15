package binlogmsg

import (
	"errors"
)

const (
	msgTableIdField      = "id"
	msgTableSubjectField = "subject"
	msgTableDataField    = "data"

	// These fields are not table columns, store them in entry for convenience only
	schemaNameField = "_s"
	tableNameField  = "_t"
	publishErrField = "_e"
)

// msgEntry are rows in msg tables.
type msgEntry map[string]interface{}

func newMsgEntry(schema, table string, entry map[string]interface{}) msgEntry {
	entry[schemaNameField] = schema
	entry[tableNameField] = table
	return msgEntry(entry)
}

func (entry msgEntry) Id() uint64 {
	return entry[msgTableIdField].(uint64)
}

func (entry msgEntry) Subject() string {
	return entry[msgTableSubjectField].(string)
}

func (entry msgEntry) Data() []byte {
	return []byte(entry[msgTableDataField].(string))
}

func (entry msgEntry) SchemaName() string {
	return entry[schemaNameField].(string)
}

func (entry msgEntry) TableName() string {
	return entry[tableNameField].(string)
}

var (
	noErr = errors.New("no error: this is just a placeholder")
)

func (entry msgEntry) SetPublishErr(err error) {
	if err == nil {
		err = noErr
	}
	entry[publishErrField] = err
}

func (entry msgEntry) GetPublishErr() error {
	err := entry[publishErrField].(error)
	if err == noErr {
		return nil
	}
	return err
}
