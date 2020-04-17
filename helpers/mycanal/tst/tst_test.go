package tst

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strings"
	"testing"
	"time"

	tstmysql "github.com/huangjunwen/tstsvc/mysql"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"

	. "github.com/huangjunwen/nproto/helpers/mycanal"
	"github.com/huangjunwen/nproto/helpers/mycanal/fulldump"
	"github.com/huangjunwen/nproto/helpers/mycanal/incrdump"
	"github.com/huangjunwen/nproto/helpers/sqlh"
)

type diffValue struct {
	ColName     string
	FullDumpVal interface{}
	IncrDumpVal interface{}
}

func TestDump(t *testing.T) {
	var err error
	assert := assert.New(t)

	var resMySQL *tstmysql.Resource
	{
		resMySQL, err = tstmysql.Run(&tstmysql.Options{
			Tag: "8.0.2",
			BaseRunOptions: dockertest.RunOptions{
				Cmd: []string{
					"--gtid-mode=ON",
					"--enforce-gtid-consistency=ON",
					"--log-bin=/var/lib/mysql/binlog",
					"--server-id=1",
					"--binlog-format=ROW",
					"--binlog-row-image=full",
					"--binlog-row-metadata=full",
				},
			},
		})
		if err != nil {
			log.Panic(err)
		}
		defer resMySQL.Close()
		log.Printf("MySQL server started.\n")
	}

	cfg := Config{
		Host:     "localhost",
		Port:     resMySQL.Options.HostPort,
		User:     "root",
		Password: resMySQL.Options.RootPassword,
	}
	fullDumpCfg := &FullDumpConfig{
		Config: cfg,
	}
	incrDumpCfg := &IncrDumpConfig{
		Config:   cfg,
		ServerId: 1001,
	}

	var db *sql.DB
	{
		db, err = resMySQL.Client()
		if err != nil {
			log.Panic(err)
		}
		defer db.Close()
		log.Printf("MySQL client created.\n")
	}

	_, err = db.Exec(`CREATE TABLE tst._types (
		id int unsigned primary key auto_increment,
		b_bit bit(64) NOT NULL DEFAULT b'0',
		n_boolean boolean not null default '0',
		n_tinyint tinyint not null default '0',
		n_smallint smallint not null default '0',
		n_mediumint mediumint not null default '0',
		n_int int not null default '0',
		n_bigint bigint not null default '0',
		n_decimal decimal(65,30) not null default '0',
		n_float float not null default '0',
		n_double double not null default '0',
		nu_tinyint tinyint unsigned not null default '0',
		nu_smallint smallint unsigned not null default '0',
		nu_mediumint mediumint unsigned not null default '0',
		nu_int int unsigned not null default '0',
		nu_bigint bigint unsigned not null default '0',
		nu_decimal decimal(65,30) unsigned not null default '0',
		nu_float float unsigned not null default '0',
		nu_double double unsigned not null default '0',
		t_year year default null,
		t_date date default null,
		t_time time default null,
		t_ftime time(6) default null,
		t_datetime datetime default null,
		t_fdatetime datetime(6) default null,
		t_timestamp timestamp default current_timestamp,
		t_ftimestamp timestamp(6) default current_timestamp(6),
		c_char char(255) not null default '',
		c_varchar varchar(255) not null default '',
		c_binary binary(64) not null default '',
		c_varbinary varbinary(64) not null default '',
		c_tinyblob tinyblob,
		c_blob blob,
		c_mediumblob mediumblob,
		c_longblob longblob,
		c_tinytext tinytext,
		c_text text,
		c_mediumtext mediumtext,
		c_longtext longtext,
		e_enum enum('a', 'b', 'c') default 'a',
		s_set set('w', 'x', 'y', 'z') default 'x',
		g_geometry geometry DEFAULT NULL,
		j_json json DEFAULT NULL
	)`)
	if err != nil {
		log.Panic(err)
	}

	colNames := []string{"id", "b_bit", "n_boolean", "n_tinyint", "n_smallint", "n_mediumint", "n_int", "n_bigint", "n_decimal", "n_float", "n_double", "nu_tinyint", "nu_smallint", "nu_mediumint", "nu_int", "nu_bigint", "nu_decimal", "nu_float", "nu_double", "t_year", "t_date", "t_time", "t_ftime", "t_datetime", "t_fdatetime", "t_timestamp", "t_ftimestamp", "c_char", "c_varchar", "c_binary", "c_varbinary", "c_tinyblob", "c_blob", "c_mediumblob", "c_longblob", "c_tinytext", "c_text", "c_mediumtext", "c_longtext", "e_enum", "s_set", "g_geometry", "j_json"}
	marks := make([]string, len(colNames))
	for i := 0; i < len(marks); i++ {
		marks[i] = "?"
	}
	query := fmt.Sprintf(
		"INSERT INTO tst._types (%s) VALUES (%s)",
		strings.Join(colNames, ", "),
		strings.Join(marks, ", "),
	)

	values := []interface{}{1, "1011", false, -128, -32768, -8388608, -2147483648, int64(-9223372036854775808), "-77777.77777", -3.1415, -2.7182, 255, 65535, 16777215, 4294967295, uint64(18446744073709551615), "8888888.8888888", 3.1415, 2.7182, "2020", "2020-04-12", "12:34:56", "12:34:56.789", "2020-04-12 12:34:56", "2020-04-21 12:34:56.789", "2020-04-21 01:23:45", "2020-04-21 01:23:45.678", "char", "varchar", "binary\x00", "varbinary\x00", nil, "blob\x00", "mediumblob\x00", "longblob\x00", nil, "text", "mediumtext", "longtext", "c", "w,y,z", nil, `{"xx":[],"a":1.223}`}
	_, err = db.Exec(query, values...)
	if err != nil {
		log.Panic(err)
	}

	var gset string
	var fullDumpVals map[string]interface{}
	gset, err = fulldump.FullDump(context.Background(), fullDumpCfg, func(ctx context.Context, q sqlh.Queryer) error {
		iter, err := fulldump.FullTableQuery(ctx, q, "tst", "_types")
		if err != nil {
			return err
		}
		defer iter(false)

		fullDumpVals, err = iter(true)
		if err != nil {
			return err
		}

		return nil
	})

	assert.NoError(err)

	_, err = db.Exec("DELETE FROM tst._types")
	assert.NoError(err)

	// Capture the deletion
	var rowDeletion *incrdump.RowDeletion
	ctx, cancel := context.WithCancel(context.Background())
	err = incrdump.IncrDump(
		ctx,
		incrDumpCfg,
		gset,
		func(ctx context.Context, e interface{}) error {
			switch ev := e.(type) {
			case *incrdump.RowDeletion:
				rowDeletion = ev
				cancel()
			}
			return nil
		},
	)
	assert.NoError(err)
	incrDumpVals := rowDeletion.BeforeDataMap()

	fmtValue := func(v interface{}) string {
		switch val := v.(type) {
		case time.Time:
			return fmt.Sprintf("time.Time<%s>", val.Format(time.RFC3339Nano))

		case string:
			return fmt.Sprintf("%+q", val)

		case nil:
			return "<nil>"

		default:
			return fmt.Sprintf("%T<%#v>", val, val)

		}
	}

	diffs := []diffValue{}

	for i, colName := range colNames {
		fullDumpVal := fullDumpVals[colName]
		incrDumpVal := incrDumpVals[colName]
		fmt.Println(
			colName,
			fmtValue(values[i]),
			fmtValue(fullDumpVal),
			fmtValue(incrDumpVal),
		)
		if !reflect.DeepEqual(fullDumpVal, incrDumpVal) {
			diffs = append(diffs, diffValue{
				ColName:     colName,
				FullDumpVal: fullDumpVal,
				IncrDumpVal: incrDumpVal,
			})
		}
	}

	fmt.Println("----------- Diff --------------")

	for _, d := range diffs {
		fmt.Println(
			d.ColName,
			fmtValue(d.FullDumpVal),
			fmtValue(d.IncrDumpVal),
		)
	}

}
