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
		query, args := dialect.DeleteStmt([]int{})
		assert.Equal("DELETE FROM `_msg_` WHERE id IN ()", query)
		assert.Empty(args)
	}
	{
		query, args := dialect.DeleteStmt([]int{1, 2, 3})
		assert.Equal("DELETE FROM `_msg_` WHERE id IN (?, ?, ?)", query)
		assert.Len(args, 3)
	}

}
