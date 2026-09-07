package data

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func IniciarBanco() {
	ctx, pool := ConectarBanco()
	defer pool.Close()
	if pool == nil || ctx == nil {
		return
	}
	defer pool.Close()

	err := criarTabelas(ctx, pool)
	if err != nil {
		log.Panicf("%s: %s", "[ERRO] Não foi possível criar as tabelas no banco de dados", err)
		return
	}
	log.Println("[LOG] Tabelas criadas no banco de dados")

	err = alimentarTabelas(ctx, pool)
	if err != nil {
		log.Panicf("%s: %s", "[ERRO] Não foi possível inserir os dados nas tabelas do banco de dados", err)
		return
	}
	log.Println("[LOG] Dados inseridos nas tabelas do banco de dados")
}

func EncerrarBanco() {
	ctx, pool := ConectarBanco()
	defer pool.Close()

	if pool == nil || ctx == nil {
		return
	}

	apagarTabelas(ctx, pool)
}

// Não encerra a conexão
// Essa responsabilidade é dada para quem usa essa função
// Para isso, usa-se defer pool.Close()
func ConectarBanco() (context.Context, *pgxpool.Pool) {
	ctx := context.Background()

	if err := godotenv.Overload(); err != nil {
		log.Println("[AVISO] Arquivo .env não encontrado")
	}

	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	dbname := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user,
		password,
		host,
		port,
		dbname,
	)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Panicf("%s: %s", "[ERRO] Não foi possível se conectar ao pool", err)
		return nil, nil
	}

	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		log.Panicf("%s: %s", "[ERRO] Não foi possível fazer ping no banco de dados", err)
		return nil, nil
	}

	return ctx, pool
}

func criarTabelas(ctx context.Context, pool *pgxpool.Pool) error {
	schema, err := os.ReadFile("data/schema.sql")

	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, string(schema))
	return err
}

func alimentarTabelas(ctx context.Context, pool *pgxpool.Pool) error {
	script, err := os.ReadFile("data/seed.sql")

	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, string(script))

	return err
}

func apagarTabelas(ctx context.Context, pool *pgxpool.Pool) error {
	script, err := os.ReadFile("data/drop.sql")

	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, string(script))

	return err
}
