package config

import (
	"flag"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNew проверяет, что New() возвращает конфиг с нулевыми значениями
func TestNew(t *testing.T) {
	want := &Config{
		Server:   Server{},   // все поля zero-value
		Postgres: Postgres{}, // все поля zero-value
	}
	got := New()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("New() = %+v, want %+v", got, want)
	}
}

// TestServer_String проверяет метод String() у Server
func TestServer_String(t *testing.T) {
	tests := []struct {
		name   string
		fields Server
		want   string
	}{
		{
			name:   "basic",
			fields: Server{Host: "127.0.0.1", Port: 8080, TimeOut: 5 * time.Second},
			want:   "127.0.0.1:8080",
		},
		{
			name:   "empty host",
			fields: Server{Host: "", Port: 1234},
			want:   ":1234",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fields.String()
			if got != tt.want {
				t.Errorf("Server.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestPostgres_String проверяет, что Postgres.String() формирует корректный DSN
func TestPostgres_String(t *testing.T) {
	tests := []struct {
		name   string
		fields Postgres
		want   string
	}{
		{
			name: "full creds",
			fields: Postgres{
				Host:     "dbhost",
				User:     "dbuser",
				Password: "dbpass",
				DBName:   "dbname",
				Port:     5432,
			},
			want: "postgres://dbuser:dbpass@dbhost:5432/dbname?sslmode=disable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fields.String()
			if got != tt.want {
				t.Errorf("Postgres.String() = %q, want %q", got, tt.want)
			}
			// дополнительно попробуем распарсить как URL
			u, err := url.Parse(got)
			assert.NoError(t, err)
			assert.Equal(t, "postgres", u.Scheme)
			assert.Equal(t, tt.fields.Host+`:`+strconv.Itoa(tt.fields.Port), u.Host)
		})
	}
}

// TestMustConfig проверяет чтение существующего файла и панику на несуществующий
func TestMustConfig(t *testing.T) {
	// 1) Успешное чтение
	content := `
server:
  host: "localhost"
  port: 9090
  timeout: 3s
postgres:
  host: "db.local"
  user: "user1"
  password: "pass1"
  dbname: "testdb"
  port: 5433
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "cfg.yml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg := MustConfig(&path)
	// Проверим пару полей
	assert := assert.New(t)
	assert.Equal("localhost", cfg.Server.Host)
	assert.Equal(9090, cfg.Server.Port)
	assert.Equal(3*time.Second, cfg.Server.TimeOut)
	assert.Equal("db.local", cfg.Postgres.Host)
	assert.Equal("user1", cfg.Postgres.User)
	assert.Equal("pass1", cfg.Postgres.Password)
	assert.Equal("testdb", cfg.Postgres.DBName)
	assert.Equal(5433, cfg.Postgres.Port)

	// 2) Файл не существует → паника
	nonExist := filepath.Join(tmpDir, "nope.yml")
	assert.False(fileExists(nonExist))
	assert.Panics(func() {
		MustConfig(&nonExist)
	})
}

// Test_fetchConfigPath проверяет логику флагов
func Test_fetchConfigPath(t *testing.T) {
	origArgs := os.Args
	origFlag := flag.CommandLine
	defer func() {
		os.Args = origArgs
		flag.CommandLine = origFlag
	}()

	// 1) Без флага → вернётся default "./config.yml"
	os.Args = []string{"cmd"} // только имя
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	got := fetchConfigPath()
	if got != "./config.yml" {
		t.Errorf("expected default ./config.yml, got %q", got)
	}

	// 2) С флагом -config
	os.Args = []string{"cmd", "-config=/my/path.yml"}
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	got2 := fetchConfigPath()
	if got2 != "/my/path.yml" {
		t.Errorf("expected /my/path.yml, got %q", got2)
	}
}

// вспомогательная функция
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
