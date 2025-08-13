package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"os"
	"sync"
	"time"
)

type Config struct {
	path        string
	ChainClient *ChainClient `yaml:"chainClient"`
	MySQL       Mysql        `yaml:"mysql"` // 数据库
	Gorm        Gorm         `yaml:"gorm"`  // gorm
}

type ChainClient struct {
	ChainId       string `json:"chainId"`
	SdkConfigPath string `json:"sdkConfigPath"`
	DefaultHeight int64  `json:"defaultHeight"`
	ContractName  string `json:"contractName"`
}

var (
	once           sync.Once
	conf           *Config
	lastChangeTime time.Time
)

func init() {
	once.Do(func() {
		conf = new(Config)
	})

	err := checkConfigEnv()
	if err != nil {
		panic(err)
	}
}

// checkConfigEnv 检擦配置环境变量是否设置
func checkConfigEnv() error {
	conf.path = os.Getenv("CONF_DIR_PATH")
	if len(conf.path) == 0 {
		return errors.New("can not find config dir path")
	}

	return nil
}

// LoadConfig 加载配置文件
func LoadConfig() error {
	viper.AddConfigPath(conf.path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("Fatal error config file: %w \n", err)
	}

	err = viper.Unmarshal(conf)
	if err != nil {
		return err
	}

	conf.ConfigFileChangeListen()

	return nil
}

// GetConfigInstance 获取配置实例
func GetConfigInstance() *Config {
	if conf != nil {
		return conf
	}
	// config 实例未初始化
	panic("config init error")
}

// 配置文件热更
func (confIns *Config) ConfigFileChangeListen() {
	viper.OnConfigChange(func(changeEvent fsnotify.Event) {
		if time.Since(lastChangeTime).Seconds() >= 1 {
			if changeEvent.Op.String() == "WRITE" {
				lastChangeTime = time.Now()
				err := viper.Unmarshal(conf)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	})
	viper.WatchConfig()
}

// mysql数据库配置
type Mysql struct {
	// ip
	Host string `json:"host" yaml:"Host"`
	// 端口
	Port int `json:"port" yaml:"Port"`
	// mysql cli用户
	User string `json:"user" yaml:"User"`
	// 密码
	Password string `json:"password" yaml:"Password"`
	// 数据库
	DBName string `json:"dbName" yaml:"DBName"`
	// 其他参数
	Parameters string `json:"parameters" yaml:"Parameters"`
}

// DSN 数据库连接串
func (m Mysql) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		m.User, m.Password, m.Host, m.Port, m.DBName, m.Parameters)
}

// Gorm 框架的相关配置
type Gorm struct {
	// 日志打印级别
	Debug bool `json:"debug" yaml:"Debug"`
	// 数据库类型：例如mysql
	DBType            string `json:"dbType" yaml:"DBType"`
	MaxLifetime       int    `json:"maxLifetime" yaml:"MaxLifetime"`
	MaxOpenConns      int    `json:"maxOpenConns" yaml:"MaxOpenConns"`
	MaxIdleConns      int    `json:"maxIdleConns" yaml:"MaxIdleConns"`
	EnableAutoMigrate bool   `json:"enableAutoMigrate" yaml:"EnableAutoMigrate"`
	// 是否开启日志打印
	IsLoggerOn bool `json:"isLoggerOn"`
}
