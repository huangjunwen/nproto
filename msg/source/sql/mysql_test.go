package sqlsrc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMySQLDialect(t *testing.T) {

	assert := assert.New(t)
	dialect := newMySQLDialect("_msg_").(*mysqlDialect)
	assert.Equal(`
			CREATE TABLE IF NOT EXISTS `+"`_msg_`"+` (
				id INT NOT NULL AUTO_INCREMENT,
				subject VARCHAR(128) NOT NULL DEFAULT "",
				data BLOB,
				PRIMARY KEY (id)
			)
		`, dialect.createStmt)
	assert.Equal("INSERT INTO `_msg_` (subject, data) VALUES (?, ?)", dialect.insertStmt)
	assert.Equal("SELECT id, subject, data FROM `_msg_` ORDER BY id", dialect.selectStmt)
	{
		assert.Equal("DELETE FROM `_msg_` WHERE id IN ()", dialect.DeleteStmt(0))
	}
	{
		assert.Equal("DELETE FROM `_msg_` WHERE id IN (?, ?, ?)", dialect.DeleteStmt(3))
	}

}
