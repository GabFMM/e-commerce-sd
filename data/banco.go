package data

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var (
	mu    sync.Mutex
	pools = map[string]*pgxpool.Pool{}
	_     = carregarEnv()
)

func carregarEnv() bool {
	if err := godotenv.Load(); err != nil {
		log.Println("[AVISO] Arquivo .env não encontrado, usando variáveis do sistema")
	}
	return true
}

// ConectarBanco abre e valida uma conexão usando as variáveis <PREFIXO>_DB_*
// prefixo "ESTOQUE" lê ESTOQUE_DB_NAME, ESTOQUE_DB_USER, ESTOQUE_DB_PASS,

func ConectarBanco(prefixo string) (*pgxpool.Pool, error) {
	ctx := context.Background()

	get := func(sufixo string) string { return os.Getenv(prefixo + "_DB_" + sufixo) }

	host := get("HOST")
	if host == "" {
		host = "localhost"
	}

	connStr := fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=disable",
		url.UserPassword(get("USER"), get("PASS")).String(),
		host, get("PORT"), get("NAME"),
	)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar pool (%s): %w", prefixo, err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("falha ao conectar no banco '%s' porta %s: %w", get("NAME"), get("PORT"), err)
	}

	log.Printf("[LOG] Conectado ao banco '%s' (porta %s)\n", get("NAME"), get("PORT"))
	return pool, nil
}

// Pool devolve um pool compartilhado por prefixo, criando-o na primeira chamada.
// Quem usa NÃO deve chamar pool.Close(). Use FecharBancos() no encerramento do MS.
func Pool(prefixo string) *pgxpool.Pool {
	mu.Lock()
	defer mu.Unlock()

	if p, ok := pools[prefixo]; ok {
		return p
	}
	p, err := ConectarBanco(prefixo)
	if err != nil {
		log.Panicf("[ERRO] %v", err)
	}
	pools[prefixo] = p
	return p
}

func FecharBancos() {
	mu.Lock()
	defer mu.Unlock()
	for k, p := range pools {
		p.Close()
		delete(pools, k)
	}
}

//Aqui eu deixei comentado pra não perderm, mas é o seguinte, agora que dividimos em containers para ter um banco
// para cada microsserviço, ao iniciar o container ele ja cria as tabelas e em alguns casos executa a seed
// então deixei só o criar conexão aqui

// package data

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"os"

// 	"github.com/jackc/pgx/v5/pgxpool"
// 	"github.com/joho/godotenv"
// )

// func IniciarBanco() {
// 	ctx, pool := ConectarBanco()
// 	defer pool.Close()
// 	if pool == nil || ctx == nil {
// 		return
// 	}
// 	defer pool.Close()

// 	err := criarTabelas(ctx, pool)
// 	if err != nil {
// 		log.Panicf("%s: %s", "[ERRO] Não foi possível criar as tabelas no banco de dados", err)
// 		return
// 	}
// 	log.Println("[LOG] Tabelas criadas no banco de dados")

// 	err = alimentarTabelas(ctx, pool)
// 	if err != nil {
// 		log.Panicf("%s: %s", "[ERRO] Não foi possível inserir os dados nas tabelas do banco de dados", err)
// 		return
// 	}
// 	log.Println("[LOG] Dados inseridos nas tabelas do banco de dados")
// }

// func EncerrarBanco() {
// 	ctx, pool := ConectarBanco()
// 	defer pool.Close()

// 	if pool == nil || ctx == nil {
// 		return
// 	}

// 	apagarTabelas(ctx, pool)
// }

// // Não encerra a conexão
// // Essa responsabilidade é dada para quem usa essa função
// // Para isso, usa-se defer pool.Close()
// func ConectarBanco() (context.Context, *pgxpool.Pool) {
// 	ctx := context.Background()

// 	if err := godotenv.Overload(); err != nil {
// 		log.Println("[AVISO] Arquivo .env não encontrado")
// 	}

// 	user := os.Getenv("DB_USER")
// 	password := os.Getenv("DB_PASSWORD")
// 	host := os.Getenv("DB_HOST")
// 	port := os.Getenv("DB_PORT")
// 	dbname := os.Getenv("DB_NAME")

// 	connStr := fmt.Sprintf(
// 		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
// 		user,
// 		password,
// 		host,
// 		port,
// 		dbname,
// 	)

// 	pool, err := pgxpool.New(ctx, connStr)
// 	if err != nil {
// 		log.Panicf("%s: %s", "[ERRO] Não foi possível se conectar ao pool", err)
// 		return nil, nil
// 	}

// 	err = pool.Ping(ctx)
// 	if err != nil {
// 		pool.Close()
// 		log.Panicf("%s: %s", "[ERRO] Não foi possível fazer ping no banco de dados", err)
// 		return nil, nil
// 	}

// 	return ctx, pool
// }

// func criarTabelas(ctx context.Context, pool *pgxpool.Pool) error {
// 	schema, err := os.ReadFile("data/schema.sql")

// 	if err != nil {
// 		return err
// 	}

// 	_, err = pool.Exec(ctx, string(schema))
// 	return err
// }

// func alimentarTabelas(ctx context.Context, pool *pgxpool.Pool) error {
// 	script, err := os.ReadFile("data/seed.sql")

// 	if err != nil {
// 		return err
// 	}

// 	_, err = pool.Exec(ctx, string(script))

// 	return err
// }

// func apagarTabelas(ctx context.Context, pool *pgxpool.Pool) error {
// 	script, err := os.ReadFile("data/drop.sql")

// 	if err != nil {
// 		return err
// 	}

// 	_, err = pool.Exec(ctx, string(script))

// 	return err
// }
