package setup

import (
	"context"
	"os"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func IniciarBanco() {
	ctx, pool := ConectarBanco()
	if pool == nil || ctx == nil {
		return
	}

	err := criarTabelas(ctx, pool);
	if err != nil {
		log.Panicf("%s: %s", "[ERRO] Não foi possível criar as tabelas no banco de dados", err)
		return
	}
	log.Println("[LOG] Tabelas criadas no banco de dados")

	err = alimentarTabelas(ctx, pool);
	if err != nil {
		log.Panicf("%s: %s", "[ERRO] Não foi possível inserir os dados nas tabelas do banco de dados", err)
		return
	}
	log.Println("[LOG] Dados inseridos nas tabelas do banco de dados")
}

func EncerrarBanco() {
	ctx, pool := ConectarBanco()
	if pool == nil || ctx == nil {
		return
	}

	apagarTabelas(ctx, pool);
}

func ConectarBanco() (context.Context, *pgxpool.Pool) {
	ctx := context.Background()
	connStr := "postgres://username:password@localhost:5432/dbname?sslmode=disable"

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Panicf("%s: %s", "[ERRO] Não foi possível se conectar ao pool", err)
		return nil, nil
	}
	defer pool.Close()

	err = pool.Ping(ctx)
	if err != nil {
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