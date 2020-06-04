package tst

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"
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
			Tag: "8.0.19",
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

		nb_bit bit(64),
		mb_bit bit(64) not null default '1011',

		ins_tinyint tinyint,
		ins_smallint smallint,
		ins_mediumint mediumint,
		ins_int int,
		ins_bigint bigint,
		ins_decimal decimal(65,30),
		ins_float float,
		ins_double double,
		inu_tinyint tinyint unsigned,
		inu_smallint smallint unsigned,
		inu_mediumint mediumint unsigned,
		inu_int int unsigned,
		inu_bigint bigint unsigned,
		inu_decimal decimal(65,30) unsigned,
		inu_float float unsigned,
		inu_double double unsigned,

		ims_tinyint tinyint not null default -128,
		ims_smallint smallint not null default -32768,
		ims_mediumint mediumint not null default -8388608,
		ims_int int not null default -2147483648,
		ims_bigint bigint not null default -9223372036854775808,
		ims_decimal decimal(65,30) not null default '-77777.77707',
		ims_float float not null default -3.1415,
		ims_double double not null default -2.7182,
		imu_tinyint tinyint unsigned not null default 255,
		imu_smallint smallint unsigned not null default 65535,
		imu_mediumint mediumint unsigned not null default 16777215,
		imu_int int unsigned not null default 4294967295,
		imu_bigint bigint unsigned not null default 18446744073709551615,
		imu_decimal decimal(65,30) unsigned not null default '88888.88808',
		imu_float float unsigned not null default 3.1415,
		imu_double double unsigned not null default 2.7182,

		tn_year year,
		tn_date date,
		tn_time time,
		tn_ftime time(6),
		tn_datetime datetime,
		tn_fdatetime datetime(6),
		tn_timestamp timestamp,
		tn_ftimestamp timestamp(6),

		tm_year year not null default '2020',
		tm_date date not null default '2020-02-20',
		tm_time time not null default '20:20:20',
		tm_ftime time(6) not null default '20:20:20.123456',
		tm_datetime datetime not null default '2020-02-20 20:20:20',
		tm_fdatetime datetime(6) not null default '2020-02-20 20:20:20.123456',
		tm_timestamp timestamp not null default current_timestamp,
		tm_ftimestamp timestamp(6) not null default current_timestamp(6),

		cn_char char(255),
		cn_varchar varchar(255),
		cn_binary binary(64),
		cn_varbinary varbinary(64),
		cn_tinyblob tinyblob,
		cn_blob blob,
		cn_mediumblob mediumblob,
		cn_longblob longblob,
		cn_tinytext tinytext,
		cn_text text,
		cn_mediumtext mediumtext,
		cn_longtext longtext,

		cm_char char(255) not null default 'char',
		cm_varchar varchar(255) not null default 'varchar',
		cm_binary binary(64) not null default 'binary\0binary\0',
		cm_varbinary varbinary(64) not null default 'varbinary\0varbinary\0',
		cm_tinyblob tinyblob default ('tinyblob\0tinyblob\0'),
		cm_blob blob default ('blob\0blob\0'),
		cm_mediumblob mediumblob default ('mediumblob\0mediumblob\0'),
		cm_longblob longblob default ('longblob\0longblob\0'),
		cm_tinytext tinytext default ('tinytext'),
		cm_text text default ('text'),
		cm_mediumtext mediumtext default ('mediumtext'),
		cm_longtext longtext default ('longtext'),

		en_enum enum('a', 'b', 'c'),
		em_enum enum('a', 'b', 'c') default 'a',

		sn_set set('w', 'x', 'y', 'z'),
		sm_set set('w', 'x', 'y', 'z') default 'w,y',

		jn_json json,
		jm_json json default ('{"a": "b",   "c":[]}'),

		g_geometry geometry default null

	)`)
	if err != nil {
		log.Panic(err)
	}

	_, err = db.Exec("INSERT INTO tst._types (id) VALUES (?)", 1)
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
	colNames := rowDeletion.ColumnNames()

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

	for _, colName := range colNames {
		fullDumpVal := fullDumpVals[colName]
		incrDumpVal := incrDumpVals[colName]
		fmt.Println(
			colName,
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
