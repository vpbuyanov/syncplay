package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Postgres Postgres `yaml:"postgres"`
	Server   Server   `yaml:"server"`
}

type Server struct {
	Host    string        `yaml:"host"`
	Port    int           `yaml:"port"`
	TimeOut time.Duration `yaml:"timeout"`
}

func (s Server) String() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type Postgres struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	Port     int    `yaml:"port"`
}

func (p Postgres) String() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(p.User, p.Password),
		Host:   fmt.Sprintf("%s:%d", p.Host, p.Port),
		Path:   p.DBName,
	}

	q := u.Query()
	q.Set("sslmode", "disable")

	u.RawQuery = q.Encode()

	return u.String()
}

func New() *Config {
	return &Config{
		Server:   Server{},
		Postgres: Postgres{},
	}
}

func MustConfig(p *string) *Config {
	var path string
	if p == nil {
		path = fetchConfigPath()
	} else {
		path = *p
	}

	if _, ok := os.Stat(path); os.IsNotExist(ok) {
		panic("Config file does not exist: " + path)
	}

	cfg := New()

	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}

	return cfg
}

func fetchConfigPath() string {
	var res string

	// --config="path/to/config.yml"
	flag.StringVar(&res, "config", "./config.yml", "path to config")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
