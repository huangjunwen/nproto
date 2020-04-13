package mycanal

import (
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/siddontang/go-mysql/replication"
)

type Config struct {
	// Host of MySQL server.
	Host string `json:"host"`

	// Port of MySQL server.
	Port uint16 `json:"port"`

	// User for connection.
	User string `json:"user"`

	// Password for connection.
	Password string `json:"password"`

	// Charset for connecting.
	Charset string `json:"charset"`
}

type FullDumpConfig struct {
	Config
}

type IncrDumpConfig struct {
	Config

	// ServerId is the server id for the slave.
	ServerId uint32 `json:"serverId"`
}

func (cfg *FullDumpConfig) ToDriverCfg() *mysql.Config {
	ret := mysql.NewConfig()
	ret.Net = "tcp"
	ret.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	ret.User = cfg.User
	ret.Passwd = cfg.Password
	ret.ParseTime = true
	ret.InterpolateParams = true
	if ret.Params == nil {
		ret.Params = map[string]string{}
	}
	ret.Params["charset"] = cfg.getCharset()
	return ret
}

func (cfg *IncrDumpConfig) ToDriverCfg() replication.BinlogSyncerConfig {
	return replication.BinlogSyncerConfig{
		ServerID:   cfg.ServerId,
		Host:       cfg.Host,
		Port:       cfg.Port,
		User:       cfg.User,
		Password:   cfg.Password,
		Charset:    cfg.getCharset(),
		ParseTime:  true,
		UseDecimal: true,
	}
}

func (cfg *Config) getCharset() string {
	if cfg.Charset != "" {
		return cfg.Charset
	}
	return "utf8mb4"
}
