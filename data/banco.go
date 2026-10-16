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
    if err := godotenv.Load("docker/.env"); err != nil {
        if errRoot := godotenv.Load(".env"); errRoot != nil {
            log.Println("[AVISO] Arquivo .env não encontrado; usando variáveis do sistema")
        }
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
